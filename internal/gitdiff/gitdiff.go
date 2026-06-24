package gitdiff

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func ChangedYAML(base string, scopes []string) ([]string, error) {
	if strings.TrimSpace(base) == "" {
		return nil, fmt.Errorf("changed-from base ref is required")
	}
	if len(scopes) == 0 {
		scopes = []string{"."}
	}

	args := []string{"diff", "--name-only", "--diff-filter=ACMR", base, "--"}
	args = append(args, scopes...)
	command := exec.Command("git", args...)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff changed files: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return FilterExistingYAML(strings.Split(string(output), "\n")), nil
}

func FilterExistingYAML(paths []string) []string {
	result := make([]string, 0, len(paths))
	seen := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" || !isYAML(path) {
			continue
		}
		if _, err := os.Stat(path); err != nil {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		result = append(result, path)
	}
	return result
}

func isYAML(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
		return true
	default:
		return false
	}
}
