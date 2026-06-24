package render

import (
	"bytes"
	"fmt"
	"os/exec"
)

type HelmOptions struct {
	Chart     string
	Release   string
	Namespace string
	Values    []string
}

type Rendered struct {
	Source  string
	Content []byte
}

func Helm(options HelmOptions) (Rendered, error) {
	if options.Chart == "" {
		return Rendered{}, fmt.Errorf("helm chart path is required")
	}
	args := helmArgs(options)
	output, err := run("helm", args...)
	if err != nil {
		return Rendered{}, err
	}
	return Rendered{Source: "helm:" + options.Chart, Content: output}, nil
}

func Kustomize(path string) (Rendered, error) {
	if path == "" {
		return Rendered{}, fmt.Errorf("kustomize path is required")
	}
	if _, err := exec.LookPath("kubectl"); err == nil {
		output, err := run("kubectl", "kustomize", path)
		if err != nil {
			return Rendered{}, err
		}
		return Rendered{Source: "kustomize:" + path, Content: output}, nil
	}
	output, err := run("kustomize", "build", path)
	if err != nil {
		return Rendered{}, err
	}
	return Rendered{Source: "kustomize:" + path, Content: output}, nil
}

func helmArgs(options HelmOptions) []string {
	release := options.Release
	if release == "" {
		release = "kube-shipguard"
	}
	args := []string{"template", release, options.Chart}
	if options.Namespace != "" {
		args = append(args, "--namespace", options.Namespace)
	}
	for _, values := range options.Values {
		args = append(args, "-f", values)
	}
	return args
}

func run(name string, args ...string) ([]byte, error) {
	command := exec.Command(name, args...)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("%s render failed: %w: %s", name, err, stderr.String())
	}
	return output, nil
}
