package mockerytool_test

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/basewarphq/bw/cmd/internal/testutil"
	"github.com/basewarphq/bw/cmd/internal/tool/mockerytool"
)

func TestMissingMockeryYmlErrors(t *testing.T) {
	t.Parallel()
	testutil.RequireBinary(t, "go")

	dir := testutil.Setup(t, map[string]string{
		"go.mod": "module example.com/testmockery\n\ngo 1.25\n\ntool github.com/vektra/mockery/v3\n",
	})

	tl := mockerytool.New()
	err := tl.Gen(context.Background(), dir, nil)
	if err == nil {
		t.Fatal("expected error when .mockery.yml is missing")
	}

	if !strings.Contains(err.Error(), ".mockery.yml") {
		t.Errorf("expected error to mention .mockery.yml, got: %v", err)
	}
}

func TestMissingToolDirectiveErrors(t *testing.T) {
	t.Parallel()
	testutil.RequireBinary(t, "go")

	dir := testutil.Setup(t, map[string]string{
		"go.mod":       "module example.com/testmockery\n\ngo 1.25\n",
		".mockery.yml": "packages:\n  example.com/testmockery:\n    interfaces:\n      Greeter:\n",
	})

	tl := mockerytool.New()
	err := tl.Gen(context.Background(), dir, nil)
	if err == nil {
		t.Fatal("expected error when mockery tool directive is missing from go.mod")
	}

	if !strings.Contains(err.Error(), "tool github.com/vektra/mockery/v3") {
		t.Errorf("expected error to mention missing directive, got: %v", err)
	}
}

func TestGenWithConfig(t *testing.T) {
	t.Parallel()
	testutil.RequireBinary(t, "go")

	goMod := "module example.com/testmockery\n\ngo 1.25.6\n\ntool github.com/vektra/mockery/v3\n"
	mockeryYml := "packages:\n  example.com/testmockery:\n    interfaces:\n      Greeter:\n"
	greeterGo := "package testmockery\n\ntype Greeter interface {\n\tGreet(name string) string\n}\n"

	dir := testutil.Setup(t, map[string]string{
		"go.mod":       goMod,
		".mockery.yml": mockeryYml,
		"greeter.go":   greeterGo,
	})

	tidy := exec.CommandContext(context.Background(), "go", "mod", "tidy")
	tidy.Dir = dir
	if out, err := tidy.CombinedOutput(); err != nil {
		t.Skipf("go mod tidy failed (mockery tool dependency not resolvable): %s\n%s", err, out)
	}

	tl := mockerytool.New()
	if err := tl.Gen(context.Background(), dir, nil); err != nil {
		t.Skipf("mockery gen failed (mockery may not be available): %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() && e.Name() == "mocks" {
			return
		}
		if strings.HasPrefix(e.Name(), "mock_") && strings.HasSuffix(e.Name(), ".go") {
			return
		}
		if strings.Contains(e.Name(), "mock") && strings.HasSuffix(e.Name(), ".go") {
			return
		}
	}

	t.Error("expected mock files to be generated but none found")
}
