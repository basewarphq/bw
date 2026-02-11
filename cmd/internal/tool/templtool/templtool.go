package templtool

import (
	"context"
	"io"
	"strings"

	"github.com/basewarphq/bw/cmd/internal/cmdexec"
	"github.com/basewarphq/bw/cmd/internal/tool"
	"github.com/cockroachdb/errors"
)

type Tool struct{}

func New() *Tool { return &Tool{} }

func (t *Tool) Name() string { return "templ" }

func (t *Tool) RunsAfter() []string { return nil }

func (t *Tool) RequiredBinaries() []tool.BinaryRequirement {
	return []tool.BinaryRequirement{
		{Name: "go", Reason: "templ is managed as a Go tool dependency"},
	}
}

func (t *Tool) RequiredFiles() []tool.FileRequirement {
	return []tool.FileRequirement{
		{
			Path:   "go.mod",
			Reason: "templ tool directive in go.mod",
			Check:  requireContains("tool github.com/a-h/templ/cmd/templ"),
		},
	}
}

func (t *Tool) Diagnose(ctx context.Context, dir string, r tool.NodeReporter) error {
	return tool.DiagnoseDefaults(ctx, dir, t, tool.BinCheckerFrom(ctx), r)
}

func (t *Tool) Gen(ctx context.Context, dir string, _ tool.NodeReporter) error {
	if err := tool.CheckFiles(dir, t.RequiredFiles()); err != nil {
		return err
	}
	return cmdexec.Run(ctx, dir, "go", "tool", "templ", "generate")
}

func (t *Tool) Lint(ctx context.Context, dir string, _ tool.NodeReporter) error {
	if err := tool.CheckFiles(dir, t.RequiredFiles()); err != nil {
		return err
	}
	return cmdexec.Run(ctx, dir, "go", "tool", "templ", "fmt", ".")
}

func requireContains(substr string) func(io.Reader) error {
	return func(rd io.Reader) error {
		data, err := io.ReadAll(rd)
		if err != nil {
			return errors.Wrap(err, "reading file")
		}
		if !strings.Contains(string(data), substr) {
			return errors.Newf("missing %q", substr)
		}
		return nil
	}
}
