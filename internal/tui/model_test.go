package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yankawai/kube-shipguard/internal/analyzer"
)

func TestFilterByHighSeverity(t *testing.T) {
	model := New([]analyzer.Finding{
		{RuleID: "KSG001", Severity: analyzer.SeverityMedium, Message: "readiness"},
		{RuleID: "KSG006", Severity: analyzer.SeverityHigh, Message: "image"},
	})

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	next := updated.(Model)

	findings := next.filteredFindings()
	if len(findings) != 1 {
		t.Fatalf("expected one high finding, got %d", len(findings))
	}
	if findings[0].RuleID != "KSG006" {
		t.Fatalf("unexpected finding: %#v", findings[0])
	}
}

func TestSearchFiltersVisibleFindings(t *testing.T) {
	model := New([]analyzer.Finding{
		{RuleID: "KSG001", Severity: analyzer.SeverityMedium, Message: "readiness probe"},
		{RuleID: "KSG006", Severity: analyzer.SeverityHigh, Message: "mutable image"},
	})
	model.query = "image"

	findings := model.filteredFindings()
	if len(findings) != 1 {
		t.Fatalf("expected one search result, got %d", len(findings))
	}
	if !strings.Contains(findings[0].Message, "image") {
		t.Fatalf("unexpected search result: %#v", findings[0])
	}
}
