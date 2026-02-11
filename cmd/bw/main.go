package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/basewarphq/bw/cmd/internal/tool"
	"github.com/basewarphq/bw/cmd/internal/tool/buftool"
	"github.com/basewarphq/bw/cmd/internal/tool/cdktool"
	"github.com/basewarphq/bw/cmd/internal/tool/goreleasertool"
	"github.com/basewarphq/bw/cmd/internal/tool/gotool"
	"github.com/basewarphq/bw/cmd/internal/tool/mockerytool"
	"github.com/basewarphq/bw/cmd/internal/tool/onepasswordtool"
	"github.com/basewarphq/bw/cmd/internal/tool/openapitool"
	"github.com/basewarphq/bw/cmd/internal/tool/shelltool"
	"github.com/basewarphq/bw/cmd/internal/tool/templtool"
	"github.com/basewarphq/bw/cmd/internal/tool/yamltool"
	"github.com/basewarphq/bw/cmd/internal/version"
	"github.com/basewarphq/bw/cmd/internal/wscfg"
)

type App struct {
	Version kong.VersionFlag `help:"Show version."`
	Project string           `short:"p" help:"Run only for a specific project (includes transitive dependencies)."`
	NoDeps  bool             `help:"With -p, skip transitive dependencies." name:"no-deps"`

	Doctor DoctorCmd `cmd:"" help:"Check that all required tools and files are present."`
	Init   InitCmd   `cmd:"" help:"Initialize local development environment."`
	Tools  struct {
		Matrix ToolsMatrixCmd `cmd:"" help:"Show the tool/step capability matrix."`
	} `cmd:"" help:"Tool commands."`

	Cdk struct {
		Bootstrap BootstrapCmd `cmd:"" help:"Bootstrap CDK in the current AWS account/region."`
		Deploy    DeployCmd    `cmd:"" help:"Deploy CDK stacks for a deployment."`
		Diff      DiffCmd      `cmd:"" help:"Show CDK diff for a deployment."`
		Slots     SlotsCmd     `cmd:"" help:"Manage dev deployment slots."`
	} `cmd:"" help:"CDK commands."`
	Build     BuildCmd     `cmd:"" help:"Build all projects."`
	Fmt       FmtCmd       `cmd:"" help:"Format code in all projects."`
	Gen       GenCmd       `cmd:"" help:"Generate code in all projects."`
	Lint      LintCmd      `cmd:"" help:"Run linters for all projects."`
	UnitTest  UnitTestCmd  `cmd:"" name:"unit-test" help:"Run unit tests for all projects."`
	Preflight PreflightCmd `cmd:"" help:"Run all doctor, gen, fmt, lint, build, and unit-test steps."`
	Release   ReleaseCmd   `cmd:"" help:"Build and publish release artifacts."`
	Infra     struct {
		Bootstrap InfraBootstrapCmd `cmd:"" help:"Bootstrap CDK in the current AWS account/region."`
		Diff      InfraDiffCmd      `cmd:"" help:"Show infrastructure diff for a deployment."`
		Deploy    InfraDeployCmd    `cmd:"" help:"Deploy infrastructure stacks for a deployment."`
		Inspect   InfraInspectCmd   `cmd:"" help:"Inspect deployment. Use -l to select lenses."`
		Slots     InfraSlotsCmd     `cmd:"" help:"Manage dev deployment slots."`
	} `cmd:"" help:"Infrastructure commands."`
}

func newRegistry() *tool.Registry {
	reg := tool.NewRegistry()
	reg.Register(templtool.New())
	reg.Register(buftool.New())
	reg.Register(openapitool.New())
	reg.Register(mockerytool.New())
	reg.Register(shelltool.New())
	reg.Register(yamltool.New())
	reg.Register(gotool.New())
	reg.Register(onepasswordtool.New())
	reg.Register(goreleasertool.New())
	reg.Register(cdktool.New())
	return reg
}

func main() {
	reg := newRegistry()

	cfg, err := wscfg.Load(reg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	var app App
	ctx := kong.Parse(&app,
		kong.Name("bw"),
		kong.Description("Basewarp development CLI."),
		kong.Vars{"version": version.Version},
		kong.Bind(cfg),
		kong.Bind(reg),
	)

	cfg.ProjectFilter = app.Project
	cfg.NoDeps = app.NoDeps

	if err := ctx.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
