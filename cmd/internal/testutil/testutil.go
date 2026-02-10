package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func Setup(tb testing.TB, files map[string]string) string {
	tb.Helper()

	root := tb.TempDir()

	for relPath, content := range files {
		fullPath := filepath.Join(root, relPath)

		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			tb.Fatalf("creating directory %s: %v", dir, err)
		}

		if err := os.WriteFile(fullPath, []byte(content), 0o600); err != nil {
			tb.Fatalf("writing file %s: %v", fullPath, err)
		}
	}

	return root
}

func RequireBinary(tb testing.TB, name string) {
	tb.Helper()

	if _, err := exec.LookPath(name); err != nil {
		tb.Skipf("skipping: %s not in PATH", name)
	}
}
