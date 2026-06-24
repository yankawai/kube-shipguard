package analyzer

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

type Severity string

const (
	SeverityLow    Severity = "low"
	SeverityMedium Severity = "medium"
	SeverityHigh   Severity = "high"
)

type Finding struct {
	RuleID      string   `json:"rule_id"`
	Severity    Severity `json:"severity"`
	Message     string   `json:"message"`
	File        string   `json:"file"`
	Kind        string   `json:"kind"`
	Namespace   string   `json:"namespace"`
	Name        string   `json:"name"`
	Container   string   `json:"container,omitempty"`
	Remediation string   `json:"remediation"`
}

func (f Finding) Location() string {
	namespace := f.Namespace
	if namespace == "" {
		namespace = "default"
	}
	return fmt.Sprintf("%s %s/%s", namespace, f.Kind, f.Name)
}

func (f Finding) Fingerprint() string {
	parts := []string{
		f.RuleID,
		f.File,
		f.Kind,
		f.Namespace,
		f.Name,
		f.Container,
		f.Message,
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:])
}

func SeverityAtLeast(actual, threshold Severity) bool {
	return severityRank(actual) >= severityRank(threshold)
}

func ParseSeverity(value string) (Severity, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "low":
		return SeverityLow, true
	case "medium", "med":
		return SeverityMedium, true
	case "high":
		return SeverityHigh, true
	default:
		return "", false
	}
}

func severityRank(severity Severity) int {
	switch severity {
	case SeverityHigh:
		return 3
	case SeverityMedium:
		return 2
	case SeverityLow:
		return 1
	default:
		return 0
	}
}
