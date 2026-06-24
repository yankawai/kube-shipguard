package baseline

import (
	"fmt"
	"os"
	"sort"

	"github.com/yankawai/kube-shipguard/internal/analyzer"
	"gopkg.in/yaml.v3"
)

const CurrentVersion = 1

type Document struct {
	Version  int     `json:"version" yaml:"version"`
	Findings []Entry `json:"findings" yaml:"findings"`
}

type Entry struct {
	Fingerprint string            `json:"fingerprint" yaml:"fingerprint"`
	RuleID      string            `json:"rule_id" yaml:"rule_id"`
	Severity    analyzer.Severity `json:"severity" yaml:"severity"`
	File        string            `json:"file" yaml:"file"`
	Kind        string            `json:"kind" yaml:"kind"`
	Namespace   string            `json:"namespace" yaml:"namespace"`
	Name        string            `json:"name" yaml:"name"`
	Container   string            `json:"container,omitempty" yaml:"container,omitempty"`
	Message     string            `json:"message" yaml:"message"`
}

type Result struct {
	Findings           []analyzer.Finding
	Suppressed         int
	BaselineEntryCount int
}

func New(findings []analyzer.Finding) Document {
	entries := make([]Entry, 0, len(findings))
	for _, finding := range findings {
		entries = append(entries, Entry{
			Fingerprint: finding.Fingerprint(),
			RuleID:      finding.RuleID,
			Severity:    finding.Severity,
			File:        finding.File,
			Kind:        finding.Kind,
			Namespace:   finding.Namespace,
			Name:        finding.Name,
			Container:   finding.Container,
			Message:     finding.Message,
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].File != entries[j].File {
			return entries[i].File < entries[j].File
		}
		if entries[i].RuleID != entries[j].RuleID {
			return entries[i].RuleID < entries[j].RuleID
		}
		return entries[i].Fingerprint < entries[j].Fingerprint
	})
	return Document{Version: CurrentVersion, Findings: entries}
}

func Write(path string, findings []analyzer.Finding) error {
	document := New(findings)
	content, err := yaml.Marshal(document)
	if err != nil {
		return fmt.Errorf("encode baseline: %w", err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return fmt.Errorf("write baseline %s: %w", path, err)
	}
	return nil
}

func Apply(path string, findings []analyzer.Finding) (Result, error) {
	if path == "" {
		return Result{Findings: findings}, nil
	}

	document, err := Load(path)
	if err != nil {
		return Result{}, err
	}

	known := make(map[string]struct{}, len(document.Findings))
	for _, entry := range document.Findings {
		if entry.Fingerprint == "" {
			continue
		}
		known[entry.Fingerprint] = struct{}{}
	}

	filtered := make([]analyzer.Finding, 0, len(findings))
	var suppressed int
	for _, finding := range findings {
		if _, ok := known[finding.Fingerprint()]; ok {
			suppressed++
			continue
		}
		filtered = append(filtered, finding)
	}

	return Result{
		Findings:           filtered,
		Suppressed:         suppressed,
		BaselineEntryCount: len(document.Findings),
	}, nil
}

func Load(path string) (Document, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Document{}, fmt.Errorf("read baseline %s: %w", path, err)
	}

	var document Document
	if err := yaml.Unmarshal(content, &document); err != nil {
		return Document{}, fmt.Errorf("parse baseline %s: %w", path, err)
	}
	if document.Version != CurrentVersion {
		return Document{}, fmt.Errorf("baseline %s uses unsupported version %d", path, document.Version)
	}
	return document, nil
}
