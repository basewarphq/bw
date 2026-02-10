package yamltool

import (
	"context"

	"github.com/basewarphq/bw/cmd/internal/cmdexec"
	"github.com/basewarphq/bw/cmd/internal/tool"
)

type Tool struct{}

func New() *Tool { return &Tool{} }

func (t *Tool) Name() string { return "yaml" }

func (t *Tool) RunsAfter() []string { return nil }

func (t *Tool) RequiredBinaries() []tool.BinaryRequirement {
	return []tool.BinaryRequirement{
		{Name: "yamlfmt", Reason: "format YAML files"},
	}
}

func (t *Tool) RequiredFiles() []tool.FileRequirement {
	return nil
}

func (t *Tool) Fmt(ctx context.Context, dir string) error {
	return cmdexec.Run(ctx, dir, "yamlfmt", ".")
}
