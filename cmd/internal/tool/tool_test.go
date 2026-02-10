package tool_test

import (
	"io"
	"strings"
	"testing"

	"github.com/basewarphq/bw/cmd/internal/testutil"
	"github.com/basewarphq/bw/cmd/internal/tool"
	"github.com/cockroachdb/errors"
)

func TestCheckFilesExistenceOnly(t *testing.T) {
	t.Parallel()

	dir := testutil.Setup(t, map[string]string{
		"go.mod": "module test\n",
	})

	reqs := []tool.FileRequirement{
		{Path: "go.mod", Reason: "Go module"},
	}

	if err := tool.CheckFiles(dir, reqs); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestCheckFilesMissingFile(t *testing.T) {
	t.Parallel()

	dir := testutil.Setup(t, map[string]string{})

	reqs := []tool.FileRequirement{
		{Path: "go.mod", Reason: "Go module"},
	}

	err := tool.CheckFiles(dir, reqs)
	if err == nil {
		t.Fatal("expected error for missing file")
	}

	if !strings.Contains(err.Error(), "go.mod") {
		t.Errorf("expected error to mention go.mod, got: %v", err)
	}
}

func TestCheckFilesCheckPasses(t *testing.T) {
	t.Parallel()

	dir := testutil.Setup(t, map[string]string{
		"go.mod": "module test\n\ntool github.com/a-h/templ/cmd/templ\n",
	})

	reqs := []tool.FileRequirement{
		{
			Path:   "go.mod",
			Reason: "templ directive",
			Check: func(rd io.Reader) error {
				data, err := io.ReadAll(rd)
				if err != nil {
					return err
				}
				if !strings.Contains(string(data), "tool github.com/a-h/templ/cmd/templ") {
					return errors.New("missing templ tool directive")
				}
				return nil
			},
		},
	}

	if err := tool.CheckFiles(dir, reqs); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestCheckFilesCheckFails(t *testing.T) {
	t.Parallel()

	dir := testutil.Setup(t, map[string]string{
		"go.mod": "module test\n\ngo 1.25\n",
	})

	reqs := []tool.FileRequirement{
		{
			Path:   "go.mod",
			Reason: "templ directive",
			Check: func(rd io.Reader) error {
				data, err := io.ReadAll(rd)
				if err != nil {
					return err
				}
				if !strings.Contains(string(data), "tool github.com/a-h/templ/cmd/templ") {
					return errors.New("missing templ tool directive")
				}
				return nil
			},
		},
	}

	err := tool.CheckFiles(dir, reqs)
	if err == nil {
		t.Fatal("expected error for failing check")
	}

	if !strings.Contains(err.Error(), "missing templ tool directive") {
		t.Errorf("expected error about missing directive, got: %v", err)
	}
}

func TestCheckFilesCheckOnMissingFile(t *testing.T) {
	t.Parallel()

	dir := testutil.Setup(t, map[string]string{})

	reqs := []tool.FileRequirement{
		{
			Path:   "go.mod",
			Reason: "templ directive",
			Check: func(_ io.Reader) error {
				return nil
			},
		},
	}

	err := tool.CheckFiles(dir, reqs)
	if err == nil {
		t.Fatal("expected error for missing file with check")
	}

	if !strings.Contains(err.Error(), "go.mod") {
		t.Errorf("expected error to mention go.mod, got: %v", err)
	}
}
