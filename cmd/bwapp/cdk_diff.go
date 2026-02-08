package main

import (
	"context"

	"github.com/basewarphq/bwapp/cmd/internal/cmdexec"
	"github.com/basewarphq/bwapp/cmd/internal/projcfg"
)

type DiffCmd struct {
	Deployment string `arg:"" required:"" help:"Deployment name (e.g., Staging, Prod)."`
}

func (c *DiffCmd) Run(cfg *projcfg.Config) error {
	return cmdexec.Run(context.Background(), cfg.CdkDir(), "cdk", "diff", "bwapp*"+c.Deployment)
}
