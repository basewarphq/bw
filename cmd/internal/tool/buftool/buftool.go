package buftool

import (
	"context"

	"github.com/basewarphq/bw/cmd/internal/cmdexec"
	"github.com/basewarphq/bw/cmd/internal/tool"
)

type Tool struct{}

func New() *Tool { return &Tool{} }

func (t *Tool) Name() string { return "buf" }

func (t *Tool) RunsAfter() []string { return nil }

func (t *Tool) RequiredBinaries() []tool.BinaryRequirement {
	return []tool.BinaryRequirement{
		{Name: "buf", Reason: "generate, format, and lint protobuf code"},
	}
}

func (t *Tool) RequiredFiles() []tool.FileRequirement {
	return []tool.FileRequirement{
		{Path: "buf.yaml", Reason: "buf module configuration"},
	}
}

func (t *Tool) Gen(ctx context.Context, dir string) error {
	if err := tool.CheckFiles(dir, t.RequiredFiles()); err != nil {
		return err
	}
	return cmdexec.Run(ctx, dir, "buf", "generate")
}

func (t *Tool) Fmt(ctx context.Context, dir string) error {
	if err := tool.CheckFiles(dir, t.RequiredFiles()); err != nil {
		return err
	}
	return cmdexec.Run(ctx, dir, "buf", "format", "-w")
}

func (t *Tool) Lint(ctx context.Context, dir string) error {
	if err := tool.CheckFiles(dir, t.RequiredFiles()); err != nil {
		return err
	}
	return cmdexec.Run(ctx, dir, "buf", "lint")
}
