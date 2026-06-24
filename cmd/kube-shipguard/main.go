package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/yankawai/kube-shipguard/internal/analyzer"
	"github.com/yankawai/kube-shipguard/internal/report"
	"github.com/yankawai/kube-shipguard/internal/scanner"
	"github.com/yankawai/kube-shipguard/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

var (
	version = "dev"
	commit  = "unknown"
)

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
	if err := flags.Parse(normalizeScanArgs(args)); err != nil {
		return err
	}

	paths := flags.Args()
	if len(paths) == 0 {
		return errors.New("scan requires at least one file or directory")
	}

	resources, err := scanner.Load(paths)
	if err != nil {
		return err
	}
	findings := analyzer.New().Analyze(resources)

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

func runReview(args []string) error {
	flags := flag.NewFlagSet("review", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	if err := flags.Parse(args); err != nil {
		return err
	}
	paths := flags.Args()
	if len(paths) == 0 {
		return errors.New("review requires at least one file or directory")
	}
	resources, err := scanner.Load(paths)
	if err != nil {
		return err
	}
	findings := analyzer.New().Analyze(resources)
	_, err = tea.NewProgram(tui.New(findings), tea.WithAltScreen()).Run()
	return err
}

func normalizeScanArgs(args []string) []string {
	flagArgs := make([]string, 0, len(args))
	pathArgs := make([]string, 0, len(args))
	for index := 0; index < len(args); index++ {
		arg := args[index]
		if arg == "--format" || arg == "--output" || arg == "--fail-on" {
			flagArgs = append(flagArgs, arg)
			if index+1 < len(args) {
				index++
				flagArgs = append(flagArgs, args[index])
			}
			continue
		}
		if strings.HasPrefix(arg, "--format=") || strings.HasPrefix(arg, "--output=") || strings.HasPrefix(arg, "--fail-on=") {
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
  kube-shipguard review <file-or-dir>...
  kube-shipguard version

Scan flags:
  --format   text, json, or sarif
  --output   output file path
  --fail-on  none, low, medium, or high`)
}
