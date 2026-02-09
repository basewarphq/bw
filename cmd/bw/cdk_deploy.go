package main

import (
	"context"

	"github.com/basewarphq/bw/cmd/internal/cdkctx"
	"github.com/basewarphq/bw/cmd/internal/cmdexec"
	"github.com/basewarphq/bw/cmd/internal/projcfg"
)

type DeployCmd struct {
	Deployment string `arg:"" optional:"" help:"Deployment name (e.g., Stag, Prod). Defaults to claimed dev slot."`
	Hotswap    bool   `help:"Enable CDK hotswap deployment for faster iterations."`
}

func (c *DeployCmd) Run(cfg *projcfg.Config) error {
	ctx := context.Background()

	deployment, err := resolveDeployment(ctx, cfg, c.Deployment)
	if err != nil {
		return err
	}

	cctx, err := cdkctx.Load(cfg.CdkDir())
	if err != nil {
		return err
	}

	args := []string{"deploy", "--require-approval", "never"}
	if c.Hotswap {
		args = append(args, "--hotswap")
	}
	args = append(args, cfg.Cdk.CdkArgs(cctx.Qualifier)...)
	args = append(args, cctx.Qualifier+"*Shared", cctx.Qualifier+"*"+deployment)
	return cmdexec.Run(ctx, cfg.CdkDir(), "cdk", args...)
}
