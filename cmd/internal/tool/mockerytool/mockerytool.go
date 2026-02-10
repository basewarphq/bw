package mockerytool

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

func (t *Tool) Name() string { return "mockery" }

func (t *Tool) Dependencies() []string { return nil }

func (t *Tool) RequiredBinaries() []tool.BinaryRequirement {
	return []tool.BinaryRequirement{
		{Name: "go", Reason: "run mockery via go tool"},
	}
}

func (t *Tool) RequiredFiles() []tool.FileRequirement {
	return []tool.FileRequirement{
		{Path: ".mockery.yml", Reason: "mockery configuration"},
		{
			Path:   "go.mod",
			Reason: "mockery tool directive in go.mod",
			Check:  requireContains("tool github.com/vektra/mockery/v3"),
		},
	}
}

func (t *Tool) Gen(ctx context.Context, dir string) error {
	if err := tool.CheckFiles(dir, t.RequiredFiles()); err != nil {
		return err
	}
	return cmdexec.Run(ctx, dir, "go", "tool", "github.com/vektra/mockery/v3")
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
