package analyzer

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/yankawai/kube-shipguard/internal/scanner"
)

var sensitiveKeyPattern = regexp.MustCompile(`(?i)(password|passwd|secret|token|apikey|api_key|private[_-]?key|client[_-]?secret)`)

type Analyzer struct{}

func New() Analyzer {
	return Analyzer{}
}

func (a Analyzer) Analyze(resources []scanner.Resource) []Finding {
	index := buildIndex(resources)
	var findings []Finding

	for _, resource := range resources {
		switch resource.Kind {
		case "Deployment", "StatefulSet", "DaemonSet", "Pod":
			findings = append(findings, analyzeWorkload(resource, index)...)
		case "Service":
			findings = append(findings, analyzeService(resource)...)
		case "Secret":
			findings = append(findings, finding(resource, "", "KSG012", SeverityHigh, "Secret manifest is stored in repository YAML", "Use External Secrets, sealed secrets, or runtime secret injection instead of committing Secret resources."))
		case "ConfigMap":
			findings = append(findings, analyzeConfigMap(resource)...)
		}
	}

	return findings
}

type resourceIndex struct {
	pdbs            []scanner.Resource
	networkPolicies []scanner.Resource
}

func buildIndex(resources []scanner.Resource) resourceIndex {
	var index resourceIndex
	for _, resource := range resources {
		switch resource.Kind {
		case "PodDisruptionBudget":
			index.pdbs = append(index.pdbs, resource)
		case "NetworkPolicy":
			index.networkPolicies = append(index.networkPolicies, resource)
		}
	}
	return index
}

func analyzeWorkload(resource scanner.Resource, index resourceIndex) []Finding {
	podSpec := podSpec(resource)
	containers := listMaps(podSpec["containers"])
	initContainers := listMaps(podSpec["initContainers"])
	allContainers := append(containers, initContainers...)
	labels := workloadLabels(resource)

	var findings []Finding
	if len(containers) == 0 {
		findings = append(findings, finding(resource, "", "KSG001", SeverityMedium, "Workload has no containers", "Define at least one application container."))
		return findings
	}

	if isReplicated(resource.Kind) && replicas(resource) < 2 {
		findings = append(findings, finding(resource, "", "KSG009", SeverityMedium, "Replicated workload has fewer than two replicas", "Run at least two replicas or document why the workload is intentionally single-instance."))
	}

	if isReplicated(resource.Kind) && !hasMatchingPDB(resource, labels, index.pdbs) {
		findings = append(findings, finding(resource, "", "KSG010", SeverityMedium, "Workload has no matching PodDisruptionBudget", "Add a PodDisruptionBudget with selector labels matching this workload."))
	}

	if !hasMatchingNetworkPolicy(resource, labels, index.networkPolicies) {
		findings = append(findings, finding(resource, "", "KSG011", SeverityMedium, "Workload has no matching NetworkPolicy", "Add a NetworkPolicy that selects this workload's pod labels."))
	}

	if workloadRunsAsRoot(resource, podSpec) {
		findings = append(findings, finding(resource, "", "KSG007", SeverityHigh, "Workload can run as root", "Set pod or container securityContext.runAsNonRoot to true."))
	}

	for _, container := range allContainers {
		containerName := stringValue(container["name"])
		if _, ok := container["readinessProbe"]; !ok && !isInitContainer(container, initContainers) {
			findings = append(findings, finding(resource, containerName, "KSG001", SeverityMedium, fmt.Sprintf("container %s has no readiness probe", containerName), "Add a readinessProbe so traffic is only sent to ready pods."))
		}
		if _, ok := container["livenessProbe"]; !ok && !isInitContainer(container, initContainers) {
			findings = append(findings, finding(resource, containerName, "KSG002", SeverityMedium, fmt.Sprintf("container %s has no liveness probe", containerName), "Add a livenessProbe to recover stuck processes."))
		}
		if !hasCompleteResources(container) {
			findings = append(findings, finding(resource, containerName, "KSG003", SeverityMedium, fmt.Sprintf("container %s has incomplete CPU/memory requests or limits", containerName), "Set cpu and memory requests and limits."))
		}
		if allowsPrivilegeEscalation(container) {
			findings = append(findings, finding(resource, containerName, "KSG004", SeverityHigh, fmt.Sprintf("container %s allows privilege escalation", containerName), "Set securityContext.allowPrivilegeEscalation to false."))
		}
		if !readOnlyRootFilesystem(container) {
			findings = append(findings, finding(resource, containerName, "KSG005", SeverityMedium, fmt.Sprintf("container %s root filesystem is writable", containerName), "Set securityContext.readOnlyRootFilesystem to true."))
		}
		if mutableImageTag(stringValue(container["image"])) {
			findings = append(findings, finding(resource, containerName, "KSG006", SeverityHigh, fmt.Sprintf("container %s uses a mutable image tag", containerName), "Pin images to immutable semantic versions or digests."))
		}
		if !dropsAllCapabilities(container) {
			findings = append(findings, finding(resource, containerName, "KSG008", SeverityMedium, fmt.Sprintf("container %s does not drop Linux capabilities", containerName), "Set securityContext.capabilities.drop to include ALL."))
		}
	}

	return findings
}

func analyzeService(resource scanner.Resource) []Finding {
	serviceType := stringValue(resource.Spec["type"])
	if serviceType == "LoadBalancer" {
		return []Finding{finding(resource, "", "KSG014", SeverityLow, "Service exposes LoadBalancer directly", "Prefer an ingress controller or document why direct LoadBalancer exposure is required.")}
	}
	return nil
}

func analyzeConfigMap(resource scanner.Resource) []Finding {
	var findings []Finding
	for key := range resource.Data {
		if sensitiveKeyPattern.MatchString(key) {
			findings = append(findings, finding(resource, "", "KSG013", SeverityHigh, fmt.Sprintf("ConfigMap key %q looks like a secret", key), "Move sensitive values to a secret manager or runtime secret injection."))
		}
	}
	return findings
}

func podSpec(resource scanner.Resource) map[string]any {
	if resource.Kind == "Pod" {
		return resource.Spec
	}
	template := mapValue(resource.Spec["template"])
	return mapValue(template["spec"])
}

func workloadLabels(resource scanner.Resource) map[string]string {
	if resource.Kind == "Pod" {
		return resource.Metadata.Labels
	}
	template := mapValue(resource.Spec["template"])
	metadata := mapValue(template["metadata"])
	return stringMap(mapValue(metadata["labels"]))
}

func replicas(resource scanner.Resource) int {
	raw, ok := resource.Spec["replicas"]
	if !ok {
		return 1
	}
	switch value := raw.(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	default:
		return 1
	}
}

func isReplicated(kind string) bool {
	return kind == "Deployment" || kind == "StatefulSet"
}

func hasMatchingPDB(resource scanner.Resource, labels map[string]string, pdbs []scanner.Resource) bool {
	for _, pdb := range pdbs {
		if pdb.Metadata.Namespace != resource.Metadata.Namespace {
			continue
		}
		selector := selectorLabels(pdb.Spec, "selector")
		if labelsMatch(selector, labels) {
			return true
		}
	}
	return false
}

func hasMatchingNetworkPolicy(resource scanner.Resource, labels map[string]string, policies []scanner.Resource) bool {
	for _, policy := range policies {
		if policy.Metadata.Namespace != resource.Metadata.Namespace {
			continue
		}
		selector := selectorLabels(policy.Spec, "podSelector")
		if labelsMatch(selector, labels) {
			return true
		}
	}
	return false
}

func selectorLabels(spec map[string]any, key string) map[string]string {
	selector := mapValue(spec[key])
	return stringMap(mapValue(selector["matchLabels"]))
}

func labelsMatch(selector, labels map[string]string) bool {
	if len(selector) == 0 || len(labels) == 0 {
		return false
	}
	for key, expected := range selector {
		if labels[key] != expected {
			return false
		}
	}
	return true
}

func workloadRunsAsRoot(resource scanner.Resource, podSpec map[string]any) bool {
	podSecurity := mapValue(podSpec["securityContext"])
	if boolValue(podSecurity["runAsNonRoot"]) {
		return false
	}

	for _, container := range listMaps(podSpec["containers"]) {
		containerSecurity := mapValue(container["securityContext"])
		if boolValue(containerSecurity["runAsNonRoot"]) {
			return false
		}
	}
	return true
}

func hasCompleteResources(container map[string]any) bool {
	resources := mapValue(container["resources"])
	requests := mapValue(resources["requests"])
	limits := mapValue(resources["limits"])
	return stringValue(requests["cpu"]) != "" &&
		stringValue(requests["memory"]) != "" &&
		stringValue(limits["cpu"]) != "" &&
		stringValue(limits["memory"]) != ""
}

func allowsPrivilegeEscalation(container map[string]any) bool {
	security := mapValue(container["securityContext"])
	value, ok := security["allowPrivilegeEscalation"]
	if !ok {
		return true
	}
	return boolValue(value)
}

func readOnlyRootFilesystem(container map[string]any) bool {
	security := mapValue(container["securityContext"])
	return boolValue(security["readOnlyRootFilesystem"])
}

func mutableImageTag(image string) bool {
	if image == "" {
		return true
	}
	if strings.Contains(image, "@sha256:") {
		return false
	}
	parts := strings.Split(image, ":")
	if len(parts) == 1 {
		return true
	}
	tag := parts[len(parts)-1]
	return tag == "" || tag == "latest"
}

func dropsAllCapabilities(container map[string]any) bool {
	security := mapValue(container["securityContext"])
	capabilities := mapValue(security["capabilities"])
	for _, value := range listValues(capabilities["drop"]) {
		if strings.EqualFold(stringValue(value), "ALL") {
			return true
		}
	}
	return false
}

func isInitContainer(container map[string]any, initContainers []map[string]any) bool {
	name := stringValue(container["name"])
	for _, initContainer := range initContainers {
		if stringValue(initContainer["name"]) == name {
			return true
		}
	}
	return false
}

func finding(resource scanner.Resource, container, ruleID string, severity Severity, message, remediation string) Finding {
	return Finding{
		RuleID:      ruleID,
		Severity:    severity,
		Message:     message,
		File:        resource.File,
		Kind:        resource.Kind,
		Namespace:   resource.Metadata.Namespace,
		Name:        resource.Metadata.Name,
		Container:   container,
		Remediation: remediation,
	}
}

func mapValue(value any) map[string]any {
	typed, ok := value.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return typed
}

func listMaps(value any) []map[string]any {
	values := listValues(value)
	result := make([]map[string]any, 0, len(values))
	for _, raw := range values {
		if typed, ok := raw.(map[string]any); ok {
			result = append(result, typed)
		}
	}
	return result
}

func listValues(value any) []any {
	typed, ok := value.([]any)
	if !ok {
		return nil
	}
	return typed
}

func stringMap(value map[string]any) map[string]string {
	result := make(map[string]string, len(value))
	for key, raw := range value {
		if text := stringValue(raw); text != "" {
			result[key] = text
		}
	}
	return result
}

func stringValue(value any) string {
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return text
}

func boolValue(value any) bool {
	typed, ok := value.(bool)
	return ok && typed
}
