package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/basewarphq/bw/cmd/internal/tool"
	"github.com/basewarphq/bw/cmd/internal/tool/buftool"
	"github.com/basewarphq/bw/cmd/internal/tool/gotool"
	"github.com/basewarphq/bw/cmd/internal/tool/mockerytool"
	"github.com/basewarphq/bw/cmd/internal/tool/openapitool"
	"github.com/basewarphq/bw/cmd/internal/tool/shelltool"
	"github.com/basewarphq/bw/cmd/internal/tool/templtool"
	"github.com/basewarphq/bw/cmd/internal/tool/yamltool"
	"github.com/basewarphq/bw/cmd/internal/version"
	"github.com/basewarphq/bw/cmd/internal/wscfg"
)

type App struct {
	Version kong.VersionFlag `help:"Show version."`
	Project string           `short:"p" help:"Run only for a specific project."`

	Doctor DoctorCmd `cmd:"" help:"Check that all required tools and files are present."`
	Init   InitCmd   `cmd:"" help:"Initialize local development environment."`
	Tools  ToolsCmd  `cmd:"" help:"Show the tool/step capability matrix."`

	Cdk struct {
		OnePasswordSync OnePasswordSyncCmd `cmd:"" name:"1psync" help:"Show 1Password sync configuration for a deployment."`
		Bootstrap       BootstrapCmd       `cmd:"" help:"Bootstrap CDK in the current AWS account/region."`
		Deploy          DeployCmd          `cmd:"" help:"Deploy CDK stacks for a deployment."`
		Diff            DiffCmd            `cmd:"" help:"Show CDK diff for a deployment."`
		Endpoints       EndpointsCmd       `cmd:"" help:"Show all gateway endpoints for a deployment."`
		LogGroups       LogGroupsCmd       `cmd:"" name:"log-groups" help:"Show all CloudWatch log groups for a deployment."`
		Slots           SlotsCmd           `cmd:"" help:"Manage dev deployment slots."`
	} `cmd:"" help:"CDK commands."`
	Check struct {
		Lint     LintCmd     `cmd:"" help:"Run linters for all projects."`
		Compiles CompilesCmd `cmd:"" help:"Check that all projects compile."`
		UnitTest UnitTestCmd `cmd:"" name:"unit-test" help:"Run unit tests for all projects."`
	} `cmd:"" help:"Check commands."`
	CheckAll CheckAllCmd `cmd:"" name:"check-all" help:"Run all dev and check steps."`
	Cli      struct {
		Release CliReleaseCmd `cmd:"" help:"Release CLI binaries using GoReleaser."`
	} `cmd:"" help:"CLI release commands."`
	Dev struct {
		Fmt FmtCmd `cmd:"" help:"Format code in all projects."`
		Gen GenCmd `cmd:"" help:"Generate code in all projects."`
	} `cmd:"" help:"Development commands."`
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
	return reg
}

func main() {
	cfg, err := wscfg.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	reg := newRegistry()

	var app App
	ctx := kong.Parse(&app,
		kong.Name("bw"),
		kong.Description("Basewarp development CLI."),
		kong.Vars{"version": version.Version},
		kong.Bind(cfg),
		kong.Bind(reg),
	)

	if app.Project != "" {
		cfg.Projects = filterProjects(cfg.Projects, app.Project)
	}

	if err := ctx.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func filterProjects(projects []wscfg.ProjectConfig, name string) []wscfg.ProjectConfig {
	for _, p := range projects {
		if p.Name == name {
			return []wscfg.ProjectConfig{p}
		}
	}
	return projects
}
