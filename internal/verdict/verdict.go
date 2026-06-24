package verdict

import "github.com/yankawai/kube-shipguard/internal/analyzer"

type Label string

const (
	LabelShip   Label = "SHIP"
	LabelReview Label = "REVIEW"
	LabelBlock  Label = "BLOCK"
)

type Evaluation struct {
	Label     Label `json:"label"`
	RiskScore int   `json:"risk_score"`
	Total     int   `json:"total"`
	High      int   `json:"high"`
	Medium    int   `json:"medium"`
	Low       int   `json:"low"`
}

func Evaluate(findings []analyzer.Finding) Evaluation {
	var evaluation Evaluation
	for _, finding := range findings {
		evaluation.Total++
		switch finding.Severity {
		case analyzer.SeverityHigh:
			evaluation.High++
			evaluation.RiskScore += 10
		case analyzer.SeverityMedium:
			evaluation.Medium++
			evaluation.RiskScore += 3
		case analyzer.SeverityLow:
			evaluation.Low++
			evaluation.RiskScore++
		}
	}

	switch {
	case evaluation.High > 0:
		evaluation.Label = LabelBlock
	case evaluation.Medium > 0 || evaluation.Low > 0:
		evaluation.Label = LabelReview
	default:
		evaluation.Label = LabelShip
	}
	return evaluation
}
