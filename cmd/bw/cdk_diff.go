package main

import (
	"context"

	"github.com/basewarphq/bw/cmd/internal/cdkctx"
	"github.com/basewarphq/bw/cmd/internal/cmdexec"
	"github.com/basewarphq/bw/cmd/internal/projcfg"
)

type DiffCmd struct {
	Deployment string `arg:"" optional:"" help:"Deployment name (e.g., Stag, Prod). Defaults to claimed dev slot."`
}

func (c *DiffCmd) Run(cfg *projcfg.Config) error {
	ctx := context.Background()

	deployment := c.Deployment
	if deployment == "" {
		claim, err := ensureClaim(ctx, cfg)
		if err != nil {
			return err
		}
		deployment = claim.Slot
	}

	cctx, err := cdkctx.Load(cfg.CdkDir())
	if err != nil {
		return err
	}

	return cmdexec.Run(ctx, cfg.CdkDir(), "cdk", "diff", cctx.Qualifier+"*"+deployment)
}
