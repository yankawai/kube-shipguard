package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yankawai/kube-shipguard/internal/analyzer"
)

func TestApplySuppressesMatchingFinding(t *testing.T) {
	path := writeConfig(t, `
suppressions:
  - rule: KSG014
    kind: Service
    namespace: default
    name: api-public
    reason: public endpoint is reviewed by platform team
    expires: 2026-12-31
`)

	result, err := Apply(path, []analyzer.Finding{serviceFinding()}, time.Date(2026, 6, 25, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("apply config: %v", err)
	}
	if result.Suppressed != 1 {
		t.Fatalf("expected 1 suppressed finding, got %d", result.Suppressed)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("expected no visible findings, got %#v", result.Findings)
	}
}

func TestApplyRejectsExpiredSuppression(t *testing.T) {
	path := writeConfig(t, `
suppressions:
  - rule: KSG014
    kind: Service
    name: api-public
    reason: temporary exception
    expires: 2026-01-01
`)

	_, err := Apply(path, []analyzer.Finding{serviceFinding()}, time.Date(2026, 6, 25, 0, 0, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("expected expired suppression error")
	}
}

func TestApplyRequiresSuppressionReason(t *testing.T) {
	path := writeConfig(t, `
suppressions:
  - rule: KSG014
    kind: Service
    name: api-public
    expires: 2026-12-31
`)

	_, err := Apply(path, []analyzer.Finding{serviceFinding()}, time.Date(2026, 6, 25, 0, 0, 0, 0, time.UTC))
	if err == nil || !strings.Contains(err.Error(), "reason") {
		t.Fatalf("expected missing reason error, got %v", err)
	}
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".kube-shipguard.yaml")
	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)+"\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func serviceFinding() analyzer.Finding {
	return analyzer.Finding{
		RuleID:    "KSG014",
		Severity:  analyzer.SeverityLow,
		Message:   "Service exposes LoadBalancer directly",
		File:      "deploy/service.yaml",
		Kind:      "Service",
		Namespace: "default",
		Name:      "api-public",
	}
}
