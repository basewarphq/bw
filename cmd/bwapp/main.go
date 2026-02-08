package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/basewarphq/bwapp/cmd/internal/projcfg"
)

type App struct {
	Cdk struct {
		OnePasswordSync OnePasswordSyncCmd `cmd:"" name:"1psync" help:"Show 1Password sync configuration for a deployment."`
		Bootstrap       BootstrapCmd       `cmd:"" help:"Bootstrap CDK in the current AWS account/region."`
		Deploy          DeployCmd          `cmd:"" help:"Deploy CDK stacks for a deployment."`
		Diff            DiffCmd            `cmd:"" help:"Show CDK diff for a deployment."`
		Endpoints       EndpointsCmd       `cmd:"" help:"Show all gateway endpoints for a deployment."`
		LogGroups       LogGroupsCmd       `cmd:"" name:"loggroups" help:"Show all CloudWatch log groups for a deployment."`
	} `cmd:"" help:"CDK commands."`
	Check struct {
		Lint     LintCmd     `cmd:"" help:"Run golangci-lint and shellcheck."`
		UnitTest UnitTestCmd `cmd:"" name:"unit-test" help:"Run all Go tests."`
	} `cmd:"" help:"Check commands."`
	Dev struct {
		Fmt FmtCmd `cmd:"" help:"Format Go files and shell scripts."`
		Gen GenCmd `cmd:"" help:"Generate code."`
	} `cmd:"" help:"Development commands."`
}

func main() {
	cfg, err := projcfg.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	var app App
	ctx := kong.Parse(&app,
		kong.Name("bwapp"),
		kong.Description("Basewarp development CLI."),
		kong.Bind(cfg),
	)
	if err := ctx.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
