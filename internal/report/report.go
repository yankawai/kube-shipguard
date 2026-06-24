package report

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/yankawai/kube-shipguard/internal/analyzer"
	"github.com/yankawai/kube-shipguard/internal/verdict"
)

type Summary struct {
	Total     int    `json:"total"`
	High      int    `json:"high"`
	Medium    int    `json:"medium"`
	Low       int    `json:"low"`
	Verdict   string `json:"verdict"`
	RiskScore int    `json:"risk_score"`
}

type JSONReport struct {
	Summary  Summary            `json:"summary"`
	Findings []analyzer.Finding `json:"findings"`
}

func WriteText(writer io.Writer, findings []analyzer.Finding) error {
	sortFindings(findings)
	evaluation := verdict.Evaluate(findings)
	if _, err := fmt.Fprintf(writer, "Kube ShipGuard verdict: %s (risk score %d)\n", evaluation.Label, evaluation.RiskScore); err != nil {
		return err
	}
	if len(findings) == 0 {
		_, err := fmt.Fprintln(writer, "Kube ShipGuard found no findings")
		return err
	}

	if _, err := fmt.Fprintf(writer, "Kube ShipGuard found %d findings\n\n", len(findings)); err != nil {
		return err
	}
	for _, finding := range findings {
		if _, err := fmt.Fprintf(
			writer,
			"%s  %s %s %s\n      %s\n",
			strings.ToUpper(string(finding.Severity)),
			finding.RuleID,
			finding.Location(),
			finding.Message,
			finding.Remediation,
		); err != nil {
			return err
		}
	}
	return nil
}

func WriteJSON(writer io.Writer, findings []analyzer.Finding) error {
	sortFindings(findings)
	if findings == nil {
		findings = []analyzer.Finding{}
	}
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(JSONReport{
		Summary:  summarize(findings),
		Findings: findings,
	})
}

func WriteSARIF(writer io.Writer, findings []analyzer.Finding) error {
	sortFindings(findings)
	evaluation := verdict.Evaluate(findings)
	rulesByID := make(map[string]sarifRule)
	results := make([]sarifResult, 0, len(findings))
	for _, finding := range findings {
		rulesByID[finding.RuleID] = sarifRule{
			ID:               finding.RuleID,
			Name:             finding.RuleID,
			ShortDescription: sarifText{Text: finding.Message},
			FullDescription:  sarifText{Text: finding.Remediation},
			Properties:       map[string]string{"severity": string(finding.Severity)},
		}
		results = append(results, sarifResult{
			RuleID:  finding.RuleID,
			Level:   sarifLevel(finding.Severity),
			Message: sarifText{Text: finding.Message},
			Locations: []sarifLocation{{
				PhysicalLocation: sarifPhysicalLocation{
					ArtifactLocation: sarifArtifactLocation{URI: finding.File},
					Region:           sarifRegion{StartLine: 1},
				},
			}},
			Properties: map[string]string{
				"kind":        finding.Kind,
				"namespace":   finding.Namespace,
				"name":        finding.Name,
				"container":   finding.Container,
				"remediation": finding.Remediation,
			},
		})
	}

	rules := make([]sarifRule, 0, len(rulesByID))
	for _, rule := range rulesByID {
		rules = append(rules, rule)
	}
	sort.Slice(rules, func(i, j int) bool { return rules[i].ID < rules[j].ID })

	doc := sarifDocument{
		Version: "2.1.0",
		Schema:  "https://json.schemastore.org/sarif-2.1.0.json",
		Runs: []sarifRun{{
			Tool: sarifTool{Driver: sarifDriver{
				Name:            "Kube ShipGuard",
				InformationURI:  "https://github.com/yankawai/kube-shipguard",
				SemanticVersion: "0.1.0",
				Rules:           rules,
			}},
			Results: results,
			Properties: map[string]any{
				"verdict":    string(evaluation.Label),
				"risk_score": evaluation.RiskScore,
			},
		}},
	}

	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(doc)
}

func summarize(findings []analyzer.Finding) Summary {
	var summary Summary
	evaluation := verdict.Evaluate(findings)
	for _, finding := range findings {
		summary.Total++
		switch finding.Severity {
		case analyzer.SeverityHigh:
			summary.High++
		case analyzer.SeverityMedium:
			summary.Medium++
		case analyzer.SeverityLow:
			summary.Low++
		}
	}
	summary.Verdict = string(evaluation.Label)
	summary.RiskScore = evaluation.RiskScore
	return summary
}

func sortFindings(findings []analyzer.Finding) {
	sort.SliceStable(findings, func(i, j int) bool {
		left := findings[i]
		right := findings[j]
		if left.File != right.File {
			return left.File < right.File
		}
		if left.RuleID != right.RuleID {
			return left.RuleID < right.RuleID
		}
		return left.Message < right.Message
	})
}

func sarifLevel(severity analyzer.Severity) string {
	switch severity {
	case analyzer.SeverityHigh:
		return "error"
	case analyzer.SeverityMedium:
		return "warning"
	default:
		return "note"
	}
}

type sarifDocument struct {
	Version string     `json:"version"`
	Schema  string     `json:"$schema"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool       sarifTool      `json:"tool"`
	Results    []sarifResult  `json:"results"`
	Properties map[string]any `json:"properties,omitempty"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name            string      `json:"name"`
	InformationURI  string      `json:"informationUri"`
	SemanticVersion string      `json:"semanticVersion"`
	Rules           []sarifRule `json:"rules"`
}

type sarifRule struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	ShortDescription sarifText         `json:"shortDescription"`
	FullDescription  sarifText         `json:"fullDescription"`
	Properties       map[string]string `json:"properties,omitempty"`
}

type sarifResult struct {
	RuleID     string            `json:"ruleId"`
	Level      string            `json:"level"`
	Message    sarifText         `json:"message"`
	Locations  []sarifLocation   `json:"locations"`
	Properties map[string]string `json:"properties,omitempty"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysicalLocation `json:"physicalLocation"`
}

type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`
	Region           sarifRegion           `json:"region"`
}

type sarifArtifactLocation struct {
	URI string `json:"uri"`
}

type sarifRegion struct {
	StartLine int `json:"startLine"`
}

type sarifText struct {
	Text string `json:"text"`
}
