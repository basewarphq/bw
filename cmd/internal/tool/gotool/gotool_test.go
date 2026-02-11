package gotool_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/basewarphq/bw/cmd/internal/testutil"
	"github.com/basewarphq/bw/cmd/internal/tool/gotool"
)

func goModContent(dir string) string {
	return "module " + dir + "\n\ngo 1.25\n"
}

const golangciConfig = `version: "2"
linters:
  default: none
  enable:
    - govet
formatters:
  enable:
    - gofmt
`

func setupGoProject(tb testing.TB) string {
	tb.Helper()
	testutil.RequireBinary(tb, "go")
	testutil.RequireBinary(tb, "golangci-lint")

	return testutil.Setup(tb, map[string]string{
		"go.mod":        goModContent("example.com/testproject"),
		".golangci.yml": golangciConfig,
	})
}

func TestFmtFormatsCode(t *testing.T) {
	t.Parallel()

	dir := setupGoProject(t)

	unformatted := `package main

import "fmt"

func main() {
fmt.Println(    "hello"   )
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(unformatted), 0o644); err != nil {
		t.Fatal(err)
	}

	tl := gotool.New()
	if err := tl.Fmt(context.Background(), dir, nil); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "main.go"))
	if err != nil {
		t.Fatal(err)
	}

	if string(got) == unformatted {
		t.Error("expected file to be reformatted, but content is unchanged")
	}
}

func TestGenRunsGoGenerate(t *testing.T) {
	t.Parallel()

	dir := setupGoProject(t)

	src := `package main

//go:generate touch generated.txt
`
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	tl := gotool.New()
	if err := tl.Gen(context.Background(), dir, nil); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, "generated.txt")); err != nil {
		t.Error("expected generated.txt to exist after go generate")
	}
}

func TestLintPassesCleanCode(t *testing.T) {
	t.Parallel()

	dir := setupGoProject(t)

	src := `package main

func main() {}
`
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	tl := gotool.New()
	if err := tl.Lint(context.Background(), dir, nil); err != nil {
		t.Errorf("expected clean code to pass lint, got: %v", err)
	}
}

func TestLintFailsBadCode(t *testing.T) {
	t.Parallel()

	dir := setupGoProject(t)

	src := `package main

func main() {
	x := 1
	_ = x
	x = 2
}
`
	cfg := `version: "2"
linters:
  default: none
  enable:
    - ineffassign
`
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".golangci.yml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}

	tl := gotool.New()
	if err := tl.Lint(context.Background(), dir, nil); err == nil {
		t.Error("expected lint to fail on code with ineffectual assignment")
	}
}

func TestBuildPassesValidCode(t *testing.T) {
	t.Parallel()

	dir := setupGoProject(t)

	src := `package main

func main() {}
`
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	tl := gotool.New()
	if err := tl.Build(context.Background(), dir, nil); err != nil {
		t.Errorf("expected valid code to build, got: %v", err)
	}
}

func TestBuildFailsSyntaxError(t *testing.T) {
	t.Parallel()

	dir := setupGoProject(t)

	src := `package main

func main() {
`
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	tl := gotool.New()
	if err := tl.Build(context.Background(), dir, nil); err == nil {
		t.Error("expected build to fail on syntax error")
	}
}

func TestUnitTestPasses(t *testing.T) {
	t.Parallel()

	dir := setupGoProject(t)

	src := `package main

func Add(a, b int) int { return a + b }
`
	testSrc := `package main

import "testing"

func TestAdd(t *testing.T) {
	if Add(1, 2) != 3 {
		t.Error("1 + 2 should be 3")
	}
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main_test.go"), []byte(testSrc), 0o644); err != nil {
		t.Fatal(err)
	}

	tl := gotool.New()
	if err := tl.UnitTest(context.Background(), dir, nil); err != nil {
		t.Errorf("expected passing test, got: %v", err)
	}
}

func TestUnitTestFails(t *testing.T) {
	t.Parallel()

	dir := setupGoProject(t)

	src := `package main

func Add(a, b int) int { return a + b }
`
	testSrc := `package main

import "testing"

func TestAdd(t *testing.T) {
	if Add(1, 2) != 999 {
		t.Error("intentional failure")
	}
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main_test.go"), []byte(testSrc), 0o644); err != nil {
		t.Fatal(err)
	}

	tl := gotool.New()
	if err := tl.UnitTest(context.Background(), dir, nil); err == nil {
		t.Error("expected failing test to return error")
	}
}

func TestMissingGoModErrors(t *testing.T) {
	t.Parallel()

	dir := testutil.Setup(t, map[string]string{
		".golangci.yml": golangciConfig,
	})

	tl := gotool.New()
	if err := tl.Fmt(context.Background(), dir, nil); err == nil {
		t.Error("expected error when go.mod is missing")
	}
}

func TestMissingGolangciYmlErrors(t *testing.T) {
	t.Parallel()

	dir := testutil.Setup(t, map[string]string{
		"go.mod": goModContent("example.com/testproject"),
	})

	tl := gotool.New()
	if err := tl.Fmt(context.Background(), dir, nil); err == nil {
		t.Error("expected error when .golangci.yml is missing")
	}
}
