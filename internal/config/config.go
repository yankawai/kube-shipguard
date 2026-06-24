package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/yankawai/kube-shipguard/internal/analyzer"
	"gopkg.in/yaml.v3"
)

type Document struct {
	Suppressions []Suppression `json:"suppressions" yaml:"suppressions"`
}

type Suppression struct {
	Rule        string `json:"rule" yaml:"rule"`
	Fingerprint string `json:"fingerprint,omitempty" yaml:"fingerprint,omitempty"`
	File        string `json:"file,omitempty" yaml:"file,omitempty"`
	Kind        string `json:"kind,omitempty" yaml:"kind,omitempty"`
	Namespace   string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	Name        string `json:"name,omitempty" yaml:"name,omitempty"`
	Container   string `json:"container,omitempty" yaml:"container,omitempty"`
	Reason      string `json:"reason" yaml:"reason"`
	Expires     string `json:"expires" yaml:"expires"`
}

type Result struct {
	Findings   []analyzer.Finding
	Suppressed int
}

func Apply(path string, findings []analyzer.Finding, now time.Time) (Result, error) {
	if path == "" {
		return Result{Findings: findings}, nil
	}

	document, err := Load(path)
	if err != nil {
		return Result{}, err
	}
	if err := validate(document, now); err != nil {
		return Result{}, err
	}

	visible := make([]analyzer.Finding, 0, len(findings))
	var suppressed int
	for _, finding := range findings {
		if isSuppressed(finding, document.Suppressions) {
			suppressed++
			continue
		}
		visible = append(visible, finding)
	}
	return Result{Findings: visible, Suppressed: suppressed}, nil
}

func Load(path string) (Document, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Document{}, fmt.Errorf("read config %s: %w", path, err)
	}

	var document Document
	if err := yaml.Unmarshal(content, &document); err != nil {
		return Document{}, fmt.Errorf("parse config %s: %w", path, err)
	}
	return document, nil
}

func validate(document Document, now time.Time) error {
	for index, suppression := range document.Suppressions {
		if strings.TrimSpace(suppression.Rule) == "" {
			return fmt.Errorf("suppression %d must set rule", index+1)
		}
		if strings.TrimSpace(suppression.Reason) == "" {
			return fmt.Errorf("suppression %d for %s must set reason", index+1, suppression.Rule)
		}
		if strings.TrimSpace(suppression.Expires) == "" {
			return fmt.Errorf("suppression %d for %s must set expires", index+1, suppression.Rule)
		}
		expires, err := time.Parse(time.DateOnly, suppression.Expires)
		if err != nil {
			return fmt.Errorf("suppression %d for %s has invalid expires date: %w", index+1, suppression.Rule, err)
		}
		if now.After(expires.Add(24 * time.Hour)) {
			return fmt.Errorf("suppression %d for %s expired on %s", index+1, suppression.Rule, suppression.Expires)
		}
	}
	return nil
}

func isSuppressed(finding analyzer.Finding, suppressions []Suppression) bool {
	for _, suppression := range suppressions {
		if !matchesSuppression(finding, suppression) {
			continue
		}
		return true
	}
	return false
}

func matchesSuppression(finding analyzer.Finding, suppression Suppression) bool {
	if suppression.Rule != finding.RuleID {
		return false
	}
	if suppression.Fingerprint != "" {
		return suppression.Fingerprint == finding.Fingerprint()
	}
	return matches(suppression.File, finding.File) &&
		matches(suppression.Kind, finding.Kind) &&
		matches(suppression.Namespace, finding.Namespace) &&
		matches(suppression.Name, finding.Name) &&
		matches(suppression.Container, finding.Container)
}

func matches(expected, actual string) bool {
	return expected == "" || expected == actual
}
