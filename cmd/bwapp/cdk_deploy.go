package main

import (
	"context"

	"github.com/basewarphq/bwapp/cmd/internal/cdkctx"
	"github.com/basewarphq/bwapp/cmd/internal/cmdexec"
	"github.com/basewarphq/bwapp/cmd/internal/projcfg"
)

type DeployCmd struct {
	Deployment string `arg:"" optional:"" help:"Deployment name (e.g., Stag, Prod). Defaults to claimed dev slot."`
	Hotswap    bool   `help:"Enable CDK hotswap deployment for faster iterations."`
}

func (c *DeployCmd) Run(cfg *projcfg.Config) error {
	ctx := context.Background()

	deployment := c.Deployment
	if deployment == "" {
		claim, err := ensureClaim(ctx, cfg)
		if err != nil {
			return err
		}
		deployment = claim.Slot
	}

	args := []string{"deploy", "--require-approval", "never"}
	if c.Hotswap {
		args = append(args, "--hotswap")
	}
	cctx, err := cdkctx.Load(cfg.CdkDir())
	if err != nil {
		return err
	}

	args = append(args, cctx.Qualifier+"*"+deployment)
	return cmdexec.Run(ctx, cfg.CdkDir(), "cdk", args...)
}
