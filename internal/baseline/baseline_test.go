package baseline

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yankawai/kube-shipguard/internal/analyzer"
)

func TestApplySuppressesKnownFindings(t *testing.T) {
	finding := testFinding("KSG006", "api")
	path := filepath.Join(t.TempDir(), "baseline.yaml")
	if err := Write(path, []analyzer.Finding{finding}); err != nil {
		t.Fatalf("write baseline: %v", err)
	}

	result, err := Apply(path, []analyzer.Finding{finding})
	if err != nil {
		t.Fatalf("apply baseline: %v", err)
	}
	if result.Suppressed != 1 {
		t.Fatalf("expected 1 suppressed finding, got %d", result.Suppressed)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("expected no visible findings, got %#v", result.Findings)
	}
}

func TestApplyKeepsNewFindings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "baseline.yaml")
	if err := Write(path, []analyzer.Finding{testFinding("KSG006", "api")}); err != nil {
		t.Fatalf("write baseline: %v", err)
	}

	result, err := Apply(path, []analyzer.Finding{
		testFinding("KSG006", "api"),
		testFinding("KSG004", "api"),
	})
	if err != nil {
		t.Fatalf("apply baseline: %v", err)
	}
	if result.Suppressed != 1 {
		t.Fatalf("expected 1 suppressed finding, got %d", result.Suppressed)
	}
	if len(result.Findings) != 1 || result.Findings[0].RuleID != "KSG004" {
		t.Fatalf("expected new KSG004 finding, got %#v", result.Findings)
	}
}

func TestLoadRejectsUnsupportedVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "baseline.yaml")
	if err := os.WriteFile(path, []byte("version: 99\nfindings: []\n"), 0o644); err != nil {
		t.Fatalf("write baseline: %v", err)
	}

	if _, err := Load(path); err == nil {
		t.Fatal("expected unsupported version error")
	}
}

func testFinding(ruleID, name string) analyzer.Finding {
	return analyzer.Finding{
		RuleID:    ruleID,
		Severity:  analyzer.SeverityHigh,
		Message:   "container api uses a mutable image tag",
		File:      "deploy/api.yaml",
		Kind:      "Deployment",
		Namespace: "default",
		Name:      name,
		Container: "api",
	}
}
