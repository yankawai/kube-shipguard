package report

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/yankawai/kube-shipguard/internal/analyzer"
)

func TestWriteJSON(t *testing.T) {
	var buffer bytes.Buffer
	err := WriteJSON(&buffer, []analyzer.Finding{{
		RuleID:   "KSG006",
		Severity: analyzer.SeverityHigh,
		Message:  "mutable image",
		File:     "deployment.yaml",
		Kind:     "Deployment",
		Name:     "api",
	}})
	if err != nil {
		t.Fatalf("write json: %v", err)
	}

	var decoded JSONReport
	if err := json.Unmarshal(buffer.Bytes(), &decoded); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if decoded.Summary.High != 1 || decoded.Summary.Total != 1 {
		t.Fatalf("unexpected summary: %#v", decoded.Summary)
	}
}

func TestWriteSARIF(t *testing.T) {
	var buffer bytes.Buffer
	err := WriteSARIF(&buffer, []analyzer.Finding{{
		RuleID:      "KSG006",
		Severity:    analyzer.SeverityHigh,
		Message:     "mutable image",
		File:        "deployment.yaml",
		Kind:        "Deployment",
		Name:        "api",
		Remediation: "pin image",
	}})
	if err != nil {
		t.Fatalf("write sarif: %v", err)
	}
	if !strings.Contains(buffer.String(), `"version": "2.1.0"`) {
		t.Fatalf("missing sarif version: %s", buffer.String())
	}
	if !strings.Contains(buffer.String(), `"ruleId": "KSG006"`) {
		t.Fatalf("missing sarif rule: %s", buffer.String())
	}
}
