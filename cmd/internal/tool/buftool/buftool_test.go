package buftool_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/basewarphq/bw/cmd/internal/testutil"
	"github.com/basewarphq/bw/cmd/internal/tool/buftool"
)

const bufYAML = "version: v2\n"

func TestFmtFormatsProto(t *testing.T) {
	t.Parallel()
	testutil.RequireBinary(t, "buf")

	dir := testutil.Setup(t, map[string]string{
		"buf.yaml": bufYAML,
	})

	unformatted := `syntax = "proto3";
package test;
message Hello {
string name = 1;
    int32    age=2;
}
`
	if err := os.WriteFile(filepath.Join(dir, "test.proto"), []byte(unformatted), 0o644); err != nil {
		t.Fatal(err)
	}

	tl := buftool.New()
	if err := tl.Fmt(context.Background(), dir); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "test.proto"))
	if err != nil {
		t.Fatal(err)
	}

	if string(got) == unformatted {
		t.Error("expected file to be reformatted, but content is unchanged")
	}
}

func TestLintPassesCleanProto(t *testing.T) {
	t.Parallel()
	testutil.RequireBinary(t, "buf")

	dir := testutil.Setup(t, map[string]string{
		"buf.yaml": bufYAML,
	})

	clean := `syntax = "proto3";
package test.v1;
message HelloRequest {
  string name = 1;
}
`
	if err := os.WriteFile(filepath.Join(dir, "test.proto"), []byte(clean), 0o644); err != nil {
		t.Fatal(err)
	}

	tl := buftool.New()
	if err := tl.Lint(context.Background(), dir); err != nil {
		t.Errorf("expected clean proto to pass lint, got: %v", err)
	}
}

func TestLintFailsBadProto(t *testing.T) {
	t.Parallel()
	testutil.RequireBinary(t, "buf")

	dir := testutil.Setup(t, map[string]string{
		"buf.yaml": bufYAML,
	})

	bad := `syntax = "proto3";
package test;
message hello_request {
  string name = 1;
}
`
	if err := os.WriteFile(filepath.Join(dir, "test.proto"), []byte(bad), 0o644); err != nil {
		t.Fatal(err)
	}

	tl := buftool.New()
	if err := tl.Lint(context.Background(), dir); err == nil {
		t.Error("expected lint to fail on proto with bad naming")
	}
}

func TestMissingBufYamlErrors(t *testing.T) {
	t.Parallel()
	testutil.RequireBinary(t, "buf")

	dir := testutil.Setup(t, map[string]string{})

	tl := buftool.New()
	err := tl.Fmt(context.Background(), dir)
	if err == nil {
		t.Fatal("expected error when buf.yaml is missing")
	}
	if !strings.Contains(err.Error(), "buf.yaml") {
		t.Errorf("expected error to mention buf.yaml, got: %v", err)
	}
}
