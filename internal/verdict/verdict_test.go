package verdict

import (
	"testing"

	"github.com/yankawai/kube-shipguard/internal/analyzer"
)

func TestEvaluateBlocksOnHighFindings(t *testing.T) {
	evaluation := Evaluate([]analyzer.Finding{
		{Severity: analyzer.SeverityHigh},
		{Severity: analyzer.SeverityMedium},
		{Severity: analyzer.SeverityLow},
	})

	if evaluation.Label != LabelBlock {
		t.Fatalf("expected block verdict, got %s", evaluation.Label)
	}
	if evaluation.RiskScore != 14 {
		t.Fatalf("expected risk score 14, got %d", evaluation.RiskScore)
	}
}

func TestEvaluateReviewsNonHighFindings(t *testing.T) {
	evaluation := Evaluate([]analyzer.Finding{{Severity: analyzer.SeverityMedium}})
	if evaluation.Label != LabelReview {
		t.Fatalf("expected review verdict, got %s", evaluation.Label)
	}
}

func TestEvaluateShipsCleanFindings(t *testing.T) {
	evaluation := Evaluate(nil)
	if evaluation.Label != LabelShip || evaluation.RiskScore != 0 {
		t.Fatalf("unexpected clean evaluation: %#v", evaluation)
	}
}
