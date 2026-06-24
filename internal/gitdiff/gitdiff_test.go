package gitdiff

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFilterExistingYAML(t *testing.T) {
	dir := t.TempDir()
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(currentDir); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	})

	if err := os.MkdirAll("deploy", 0o755); err != nil {
		t.Fatalf("mkdir deploy: %v", err)
	}
	for _, path := range []string{"deploy/api.yaml", "deploy/service.yml", "README.md"} {
		if err := os.WriteFile(filepath.Clean(path), []byte("x"), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	got := FilterExistingYAML([]string{
		"deploy/api.yaml",
		"deploy/api.yaml",
		"deploy/service.yml",
		"deploy/missing.yaml",
		"README.md",
	})
	if len(got) != 2 {
		t.Fatalf("expected 2 yaml paths, got %#v", got)
	}
}
