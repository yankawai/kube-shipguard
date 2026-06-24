package analyzer

import (
	"testing"

	"github.com/yankawai/kube-shipguard/internal/scanner"
)

func TestAnalyzeSecureManifestHasNoFindings(t *testing.T) {
	resources, err := scanner.Load([]string{"../../examples/secure"})
	if err != nil {
		t.Fatalf("load secure manifests: %v", err)
	}

	findings := New().Analyze(resources)
	if len(findings) != 0 {
		t.Fatalf("expected no findings, got %d: %#v", len(findings), findings)
	}
}

func TestAnalyzeUnsafeManifestFindsHighRiskIssues(t *testing.T) {
	resources, err := scanner.Load([]string{"../../examples/unsafe"})
	if err != nil {
		t.Fatalf("load unsafe manifests: %v", err)
	}

	findings := New().Analyze(resources)
	rules := make(map[string]bool)
	for _, finding := range findings {
		rules[finding.RuleID] = true
	}

	for _, expected := range []string{"KSG004", "KSG006", "KSG007", "KSG012", "KSG013"} {
		if !rules[expected] {
			t.Fatalf("missing expected rule %s in findings: %#v", expected, findings)
		}
	}
}

func TestFindingFingerprintIsStableForSameFinding(t *testing.T) {
	finding := Finding{
		RuleID:    "KSG006",
		Severity:  SeverityHigh,
		Message:   "container api uses a mutable image tag",
		File:      "deploy/api.yaml",
		Kind:      "Deployment",
		Namespace: "default",
		Name:      "api",
		Container: "api",
	}

	copy := finding
	if finding.Fingerprint() == "" {
		t.Fatal("expected non-empty fingerprint")
	}
	if finding.Fingerprint() != copy.Fingerprint() {
		t.Fatal("expected equal findings to have equal fingerprints")
	}

	copy.Container = "worker"
	if finding.Fingerprint() == copy.Fingerprint() {
		t.Fatal("expected changed finding identity to change fingerprint")
	}
}
