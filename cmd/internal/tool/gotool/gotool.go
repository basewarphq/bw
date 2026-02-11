package gotool

import (
	"context"

	"github.com/basewarphq/bw/cmd/internal/cmdexec"
	"github.com/basewarphq/bw/cmd/internal/tool"
)

type Tool struct{}

func New() *Tool { return &Tool{} }

func (t *Tool) Name() string { return "go" }

func (t *Tool) RunsAfter() []string { return []string{"templ"} }

func (t *Tool) RequiredBinaries() []tool.BinaryRequirement {
	return []tool.BinaryRequirement{
		{Name: "go", Reason: "build, generate, and test Go code"},
		{Name: "golangci-lint", Reason: "format and lint Go code"},
	}
}

func (t *Tool) RequiredFiles() []tool.FileRequirement {
	return []tool.FileRequirement{
		{Path: "go.mod", Reason: "Go module definition"},
		{Path: ".golangci.yml", Reason: "golangci-lint configuration"},
	}
}

func (t *Tool) Diagnose(ctx context.Context, dir string, r tool.NodeReporter) error {
	return tool.DiagnoseDefaults(ctx, dir, t, tool.BinCheckerFrom(ctx), r)
}

func (t *Tool) Init(ctx context.Context, dir string, _ tool.NodeReporter) error {
	if err := tool.CheckFiles(dir, t.RequiredFiles()); err != nil {
		return err
	}
	return cmdexec.Run(ctx, dir, "go", "mod", "download")
}

func (t *Tool) Fmt(ctx context.Context, dir string, _ tool.NodeReporter) error {
	if err := tool.CheckFiles(dir, t.RequiredFiles()); err != nil {
		return err
	}
	if err := cmdexec.Run(ctx, dir, "go", "mod", "tidy"); err != nil {
		return err
	}
	return cmdexec.Run(ctx, dir, "golangci-lint", "fmt", "./...")
}

func (t *Tool) Gen(ctx context.Context, dir string, _ tool.NodeReporter) error {
	if err := tool.CheckFiles(dir, t.RequiredFiles()); err != nil {
		return err
	}
	return cmdexec.Run(ctx, dir, "go", "generate", "./...")
}

func (t *Tool) Lint(ctx context.Context, dir string, _ tool.NodeReporter) error {
	if err := tool.CheckFiles(dir, t.RequiredFiles()); err != nil {
		return err
	}
	return cmdexec.Run(ctx, dir, "golangci-lint", "run", "./...")
}

func (t *Tool) Build(ctx context.Context, dir string, _ tool.NodeReporter) error {
	if err := tool.CheckFiles(dir, t.RequiredFiles()); err != nil {
		return err
	}
	return cmdexec.Run(ctx, dir, "go", "build", "./...")
}

func (t *Tool) UnitTest(ctx context.Context, dir string, _ tool.NodeReporter) error {
	if err := tool.CheckFiles(dir, t.RequiredFiles()); err != nil {
		return err
	}
	return cmdexec.Run(ctx, dir, "go", "test", "./...")
}
