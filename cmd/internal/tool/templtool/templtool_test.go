package templtool_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/basewarphq/bw/cmd/internal/testutil"
	"github.com/basewarphq/bw/cmd/internal/tool/templtool"
)

func setupTemplProject(t *testing.T) string {
	t.Helper()
	testutil.RequireBinary(t, "go")

	goMod := "module example.com/testtempl\n\ngo 1.25\n\ntool github.com/a-h/templ/cmd/templ\n\nrequire github.com/a-h/templ v0.3.977\n"
	templFile := "package main\n\ntempl Hello() {\n\t<p>Hello</p>\n}\n"

	dir := testutil.Setup(t, map[string]string{
		"go.mod":      goMod,
		"hello.templ": templFile,
	})

	cmd := exec.CommandContext(context.Background(), "go", "mod", "tidy")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("skipping: go mod tidy failed (templ may not be available): %s\n%s", err, out)
	}

	return dir
}

func TestGenGeneratesTemplFile(t *testing.T) {
	t.Parallel()
	dir := setupTemplProject(t)

	tl := templtool.New()
	if err := tl.Gen(context.Background(), dir); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, "hello_templ.go")); err != nil {
		t.Error("expected hello_templ.go to be generated")
	}
}

func TestGenMissingToolDirectiveErrors(t *testing.T) {
	t.Parallel()
	testutil.RequireBinary(t, "go")

	dir := testutil.Setup(t, map[string]string{
		"go.mod":      "module example.com/testtempl\n\ngo 1.25\n",
		"hello.templ": "package main\n\ntempl Hello() {\n\t<p>Hello</p>\n}\n",
	})

	tl := templtool.New()
	err := tl.Gen(context.Background(), dir)
	if err == nil {
		t.Fatal("expected error when templ tool directive is missing from go.mod")
	}

	if !strings.Contains(err.Error(), "tool github.com/a-h/templ/cmd/templ") {
		t.Errorf("expected error to mention missing directive, got: %v", err)
	}
}
