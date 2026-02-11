package shelltool_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/basewarphq/bw/cmd/internal/testutil"
	"github.com/basewarphq/bw/cmd/internal/tool/shelltool"
)

func TestFmtFormatsScript(t *testing.T) {
	t.Parallel()
	testutil.RequireBinary(t, "shfmt")

	unformatted := `#!/usr/bin/env bash
if [   true   ]; then
echo "hello"
fi
`
	dir := testutil.Setup(t, map[string]string{
		"script.sh": unformatted,
	})

	tl := shelltool.New()
	if err := tl.Fmt(context.Background(), dir, nil); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "script.sh"))
	if err != nil {
		t.Fatal(err)
	}

	if string(got) == unformatted {
		t.Error("expected file to be reformatted, but content is unchanged")
	}
}

func TestFmtNoShellScripts(t *testing.T) {
	t.Parallel()
	testutil.RequireBinary(t, "shfmt")

	dir := testutil.Setup(t, map[string]string{
		"readme.md": "# hello",
	})

	tl := shelltool.New()
	if err := tl.Fmt(context.Background(), dir, nil); err != nil {
		t.Errorf("expected no error when no shell scripts present, got: %v", err)
	}
}

func TestLintPassesCleanScript(t *testing.T) {
	t.Parallel()
	testutil.RequireBinary(t, "shellcheck")

	clean := `#!/usr/bin/env bash
set -euo pipefail

echo "hello"
`
	dir := testutil.Setup(t, map[string]string{
		"script.sh": clean,
	})

	tl := shelltool.New()
	if err := tl.Lint(context.Background(), dir, nil); err != nil {
		t.Errorf("expected clean script to pass lint, got: %v", err)
	}
}

func TestLintFailsBadScript(t *testing.T) {
	t.Parallel()
	testutil.RequireBinary(t, "shellcheck")

	bad := `#!/usr/bin/env bash
echo $UNQUOTED_VAR
`
	dir := testutil.Setup(t, map[string]string{
		"script.sh": bad,
	})

	tl := shelltool.New()
	if err := tl.Lint(context.Background(), dir, nil); err == nil {
		t.Error("expected lint to fail on unquoted variable")
	}
}

func TestLintNoShellScripts(t *testing.T) {
	t.Parallel()
	testutil.RequireBinary(t, "shellcheck")

	dir := testutil.Setup(t, map[string]string{
		"readme.md": "# hello",
	})

	tl := shelltool.New()
	if err := tl.Lint(context.Background(), dir, nil); err != nil {
		t.Errorf("expected no error when no shell scripts present, got: %v", err)
	}
}
