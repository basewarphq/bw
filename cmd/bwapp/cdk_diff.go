package main

import (
	"context"

	"github.com/basewarphq/bwapp/cmd/internal/cmdexec"
	"github.com/basewarphq/bwapp/cmd/internal/projcfg"
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

	return cmdexec.Run(ctx, cfg.CdkDir(), "cdk", "diff", "bwapp*"+deployment)
}
