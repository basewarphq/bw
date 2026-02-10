package yamltool_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/basewarphq/bw/cmd/internal/testutil"
	"github.com/basewarphq/bw/cmd/internal/tool/yamltool"
)

func TestFmtFormatsYaml(t *testing.T) {
	t.Parallel()
	testutil.RequireBinary(t, "yamlfmt")

	unformatted := `name:    hello
items:
  -  one
  -    two
  -  three
`
	dir := testutil.Setup(t, map[string]string{
		"test.yaml": unformatted,
	})

	tl := yamltool.New()
	if err := tl.Fmt(context.Background(), dir); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "test.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	if string(got) == unformatted {
		t.Error("expected file to be reformatted, but content is unchanged")
	}
}

func TestFmtNoOpOnCleanYaml(t *testing.T) {
	t.Parallel()
	testutil.RequireBinary(t, "yamlfmt")

	clean := `name: hello
items:
  - one
  - two
  - three
`
	dir := testutil.Setup(t, map[string]string{
		"test.yaml": clean,
	})

	tl := yamltool.New()
	if err := tl.Fmt(context.Background(), dir); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "test.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	if string(got) != clean {
		t.Errorf("expected file to remain unchanged, got:\n%s", string(got))
	}
}
