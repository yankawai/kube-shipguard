package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/yankawai/kube-shipguard/internal/analyzer"
	"github.com/yankawai/kube-shipguard/internal/baseline"
	"github.com/yankawai/kube-shipguard/internal/config"
	"github.com/yankawai/kube-shipguard/internal/gitdiff"
	"github.com/yankawai/kube-shipguard/internal/render"
	"github.com/yankawai/kube-shipguard/internal/report"
	"github.com/yankawai/kube-shipguard/internal/scanner"
	"github.com/yankawai/kube-shipguard/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

var (
	version = "dev"
	commit  = "unknown"
)

type multiFlag []string

func (m *multiFlag) String() string {
	return strings.Join(*m, ",")
}

func (m *multiFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}

type inputOptions struct {
	Paths         []string
	ChangedFrom   string
	HelmChart     string
	HelmRelease   string
	HelmNamespace string
	HelmValues    []string
	KustomizePath string
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		printUsage(stderr)
		return errors.New("missing command")
	}

	switch args[0] {
	case "scan":
		return runScan(args[1:], stdout)
	case "baseline":
		return runBaseline(args[1:], stdout)
	case "review":
		return runReview(args[1:])
	case "version":
		fmt.Fprintf(stdout, "kube-shipguard %s (%s)\n", version, commit)
		return nil
	case "help", "-h", "--help":
		printUsage(stdout)
		return nil
	default:
		printUsage(stderr)
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runScan(args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("scan", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	format := flags.String("format", "text", "output format: text, json, sarif")
	output := flags.String("output", "", "write output to file")
	failOn := flags.String("fail-on", "high", "minimum severity that fails: none, low, medium, high")
	baselinePath := flags.String("baseline", "", "ignore findings already captured in a baseline file")
	configPath := flags.String("config", "", "configuration file with expiring suppressions")
	changedFrom := flags.String("changed-from", "", "only scan YAML files changed from the git ref or range")
	helmChart := flags.String("helm-chart", "", "render a Helm chart before scanning")
	helmRelease := flags.String("helm-release", "", "Helm release name for rendered manifests")
	helmNamespace := flags.String("helm-namespace", "", "Helm namespace for rendered manifests")
	var helmValues multiFlag
	flags.Var(&helmValues, "helm-values", "Helm values file; may be repeated")
	kustomizePath := flags.String("kustomize", "", "render a Kustomize directory before scanning")
	if err := flags.Parse(normalizeScanArgs(args)); err != nil {
		return err
	}

	paths := flags.Args()
	if len(paths) == 0 && *changedFrom == "" && *helmChart == "" && *kustomizePath == "" {
		return errors.New("scan requires at least one file or directory")
	}

	resources, err := loadInputs(inputOptions{
		Paths:         paths,
		ChangedFrom:   *changedFrom,
		HelmChart:     *helmChart,
		HelmRelease:   *helmRelease,
		HelmNamespace: *helmNamespace,
		HelmValues:    helmValues,
		KustomizePath: *kustomizePath,
	})
	if err != nil {
		return err
	}
	findings := analyzer.New().Analyze(resources)
	configResult, err := config.Apply(*configPath, findings, time.Now())
	if err != nil {
		return err
	}
	findings = configResult.Findings
	baselineResult, err := baseline.Apply(*baselinePath, findings)
	if err != nil {
		return err
	}
	findings = baselineResult.Findings

	writer := stdout
	var file *os.File
	if *output != "" {
		file, err = os.Create(*output)
		if err != nil {
			return fmt.Errorf("create output file: %w", err)
		}
		defer file.Close()
		writer = file
	}

	switch *format {
	case "text":
		if configResult.Suppressed > 0 {
			if _, err := fmt.Fprintf(writer, "Config suppressed %d accepted findings\n\n", configResult.Suppressed); err != nil {
				return err
			}
		}
		if baselineResult.Suppressed > 0 {
			if _, err := fmt.Fprintf(writer, "Baseline suppressed %d known findings\n\n", baselineResult.Suppressed); err != nil {
				return err
			}
		}
		err = report.WriteText(writer, findings)
	case "json":
		err = report.WriteJSON(writer, findings)
	case "sarif":
		err = report.WriteSARIF(writer, findings)
	default:
		return fmt.Errorf("unsupported format %q", *format)
	}
	if err != nil {
		return err
	}

	if shouldFail(findings, *failOn) {
		return fmt.Errorf("findings met fail-on threshold %q", *failOn)
	}
	return nil
}

func loadInputs(options inputOptions) ([]scanner.Resource, error) {
	paths := options.Paths
	var err error
	if options.ChangedFrom != "" {
		paths, err = gitdiff.ChangedYAML(options.ChangedFrom, paths)
		if err != nil {
			return nil, err
		}
	}

	var resources []scanner.Resource
	if len(paths) > 0 {
		loaded, err := scanner.Load(paths)
		if err != nil {
			return nil, err
		}
		resources = append(resources, loaded...)
	}

	if options.HelmChart != "" {
		rendered, err := render.Helm(render.HelmOptions{
			Chart:     options.HelmChart,
			Release:   options.HelmRelease,
			Namespace: options.HelmNamespace,
			Values:    options.HelmValues,
		})
		if err != nil {
			return nil, err
		}
		loaded, err := scanner.LoadYAML(rendered.Source, rendered.Content)
		if err != nil {
			return nil, err
		}
		resources = append(resources, loaded...)
	}

	if options.KustomizePath != "" {
		rendered, err := render.Kustomize(options.KustomizePath)
		if err != nil {
			return nil, err
		}
		loaded, err := scanner.LoadYAML(rendered.Source, rendered.Content)
		if err != nil {
			return nil, err
		}
		resources = append(resources, loaded...)
	}

	return resources, nil
}

func runBaseline(args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("baseline", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	output := flags.String("output", ".kube-shipguard-baseline.yaml", "baseline output file")
	helmChart := flags.String("helm-chart", "", "render a Helm chart before scanning")
	helmRelease := flags.String("helm-release", "", "Helm release name for rendered manifests")
	helmNamespace := flags.String("helm-namespace", "", "Helm namespace for rendered manifests")
	var helmValues multiFlag
	flags.Var(&helmValues, "helm-values", "Helm values file; may be repeated")
	kustomizePath := flags.String("kustomize", "", "render a Kustomize directory before scanning")
	if err := flags.Parse(normalizeBaselineArgs(args)); err != nil {
		return err
	}

	paths := flags.Args()
	if len(paths) == 0 && *helmChart == "" && *kustomizePath == "" {
		return errors.New("baseline requires at least one file or directory")
	}

	resources, err := loadInputs(inputOptions{
		Paths:         paths,
		HelmChart:     *helmChart,
		HelmRelease:   *helmRelease,
		HelmNamespace: *helmNamespace,
		HelmValues:    helmValues,
		KustomizePath: *kustomizePath,
	})
	if err != nil {
		return err
	}
	findings := analyzer.New().Analyze(resources)
	if err := baseline.Write(*output, findings); err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "Wrote baseline with %d findings to %s\n", len(findings), *output)
	return err
}

func runReview(args []string) error {
	flags := flag.NewFlagSet("review", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	configPath := flags.String("config", "", "configuration file with expiring suppressions")
	helmChart := flags.String("helm-chart", "", "render a Helm chart before scanning")
	helmRelease := flags.String("helm-release", "", "Helm release name for rendered manifests")
	helmNamespace := flags.String("helm-namespace", "", "Helm namespace for rendered manifests")
	var helmValues multiFlag
	flags.Var(&helmValues, "helm-values", "Helm values file; may be repeated")
	kustomizePath := flags.String("kustomize", "", "render a Kustomize directory before scanning")
	if err := flags.Parse(normalizeReviewArgs(args)); err != nil {
		return err
	}
	paths := flags.Args()
	if len(paths) == 0 && *helmChart == "" && *kustomizePath == "" {
		return errors.New("review requires at least one file or directory")
	}
	resources, err := loadInputs(inputOptions{
		Paths:         paths,
		HelmChart:     *helmChart,
		HelmRelease:   *helmRelease,
		HelmNamespace: *helmNamespace,
		HelmValues:    helmValues,
		KustomizePath: *kustomizePath,
	})
	if err != nil {
		return err
	}
	findings := analyzer.New().Analyze(resources)
	configResult, err := config.Apply(*configPath, findings, time.Now())
	if err != nil {
		return err
	}
	findings = configResult.Findings
	_, err = tea.NewProgram(tui.New(findings), tea.WithAltScreen()).Run()
	return err
}

func normalizeScanArgs(args []string) []string {
	flagArgs := make([]string, 0, len(args))
	pathArgs := make([]string, 0, len(args))
	for index := 0; index < len(args); index++ {
		arg := args[index]
		if arg == "--format" || arg == "--output" || arg == "--fail-on" || arg == "--baseline" || arg == "--config" || arg == "--changed-from" || arg == "--helm-chart" || arg == "--helm-release" || arg == "--helm-namespace" || arg == "--helm-values" || arg == "--kustomize" {
			flagArgs = append(flagArgs, arg)
			if index+1 < len(args) {
				index++
				flagArgs = append(flagArgs, args[index])
			}
			continue
		}
		if strings.HasPrefix(arg, "--format=") || strings.HasPrefix(arg, "--output=") || strings.HasPrefix(arg, "--fail-on=") || strings.HasPrefix(arg, "--baseline=") || strings.HasPrefix(arg, "--config=") || strings.HasPrefix(arg, "--changed-from=") || strings.HasPrefix(arg, "--helm-chart=") || strings.HasPrefix(arg, "--helm-release=") || strings.HasPrefix(arg, "--helm-namespace=") || strings.HasPrefix(arg, "--helm-values=") || strings.HasPrefix(arg, "--kustomize=") {
			flagArgs = append(flagArgs, arg)
			continue
		}
		pathArgs = append(pathArgs, arg)
	}
	return append(flagArgs, pathArgs...)
}

func normalizeBaselineArgs(args []string) []string {
	flagArgs := make([]string, 0, len(args))
	pathArgs := make([]string, 0, len(args))
	for index := 0; index < len(args); index++ {
		arg := args[index]
		if arg == "--output" || arg == "--helm-chart" || arg == "--helm-release" || arg == "--helm-namespace" || arg == "--helm-values" || arg == "--kustomize" {
			flagArgs = append(flagArgs, arg)
			if index+1 < len(args) {
				index++
				flagArgs = append(flagArgs, args[index])
			}
			continue
		}
		if strings.HasPrefix(arg, "--output=") || strings.HasPrefix(arg, "--helm-chart=") || strings.HasPrefix(arg, "--helm-release=") || strings.HasPrefix(arg, "--helm-namespace=") || strings.HasPrefix(arg, "--helm-values=") || strings.HasPrefix(arg, "--kustomize=") {
			flagArgs = append(flagArgs, arg)
			continue
		}
		pathArgs = append(pathArgs, arg)
	}
	return append(flagArgs, pathArgs...)
}

func normalizeReviewArgs(args []string) []string {
	flagArgs := make([]string, 0, len(args))
	pathArgs := make([]string, 0, len(args))
	for index := 0; index < len(args); index++ {
		arg := args[index]
		if arg == "--config" || arg == "--helm-chart" || arg == "--helm-release" || arg == "--helm-namespace" || arg == "--helm-values" || arg == "--kustomize" {
			flagArgs = append(flagArgs, arg)
			if index+1 < len(args) {
				index++
				flagArgs = append(flagArgs, args[index])
			}
			continue
		}
		if strings.HasPrefix(arg, "--config=") || strings.HasPrefix(arg, "--helm-chart=") || strings.HasPrefix(arg, "--helm-release=") || strings.HasPrefix(arg, "--helm-namespace=") || strings.HasPrefix(arg, "--helm-values=") || strings.HasPrefix(arg, "--kustomize=") {
			flagArgs = append(flagArgs, arg)
			continue
		}
		pathArgs = append(pathArgs, arg)
	}
	return append(flagArgs, pathArgs...)
}

func shouldFail(findings []analyzer.Finding, threshold string) bool {
	if threshold == "none" {
		return false
	}
	severity, ok := analyzer.ParseSeverity(threshold)
	if !ok {
		return true
	}
	for _, finding := range findings {
		if analyzer.SeverityAtLeast(finding.Severity, severity) {
			return true
		}
	}
	return false
}

func printUsage(writer io.Writer) {
	fmt.Fprintln(writer, `Kube ShipGuard

Usage:
  kube-shipguard scan [flags] <file-or-dir>...
  kube-shipguard baseline [flags] <file-or-dir>...
  kube-shipguard review <file-or-dir>...
  kube-shipguard version

Scan flags:
  --format   text, json, or sarif
  --output   output file path
  --fail-on  none, low, medium, or high
  --baseline ignore known findings from a baseline file
  --config   config file with expiring suppressions
  --changed-from only scan YAML files changed from a git ref or range
  --helm-chart render a Helm chart before scanning
  --helm-values Helm values file; may be repeated
  --kustomize render a Kustomize directory before scanning

Baseline flags:
  --output   baseline output file

Render flags:
  --helm-chart render a Helm chart before scanning
  --helm-release Helm release name
  --helm-namespace Helm namespace
  --helm-values Helm values file; may be repeated
  --kustomize render a Kustomize directory before scanning`)
}
