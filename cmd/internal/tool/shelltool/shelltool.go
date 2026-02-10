package shelltool

import (
	"context"

	"github.com/basewarphq/bw/cmd/internal/cmdexec"
	"github.com/basewarphq/bw/cmd/internal/shellfiles"
	"github.com/basewarphq/bw/cmd/internal/tool"
)

type Tool struct{}

func New() *Tool { return &Tool{} }

func (t *Tool) Name() string { return "shell" }

func (t *Tool) Dependencies() []string { return nil }

func (t *Tool) RequiredBinaries() []tool.BinaryRequirement {
	return []tool.BinaryRequirement{
		{Name: "shfmt", Reason: "format shell scripts"},
		{Name: "shellcheck", Reason: "lint shell scripts"},
	}
}

func (t *Tool) RequiredFiles() []tool.FileRequirement {
	return nil
}

func (t *Tool) Fmt(ctx context.Context, dir string) error {
	scripts, err := shellfiles.FindShellScripts(dir)
	if err != nil {
		return err
	}
	if len(scripts) == 0 {
		return nil
	}
	args := append([]string{"-w"}, scripts...)
	return cmdexec.Run(ctx, dir, "shfmt", args...)
}

func (t *Tool) Lint(ctx context.Context, dir string) error {
	scripts, err := shellfiles.FindShellScripts(dir)
	if err != nil {
		return err
	}
	if len(scripts) == 0 {
		return nil
	}
	return cmdexec.Run(ctx, dir, "shellcheck", scripts...)
}
