package main

import (
	"context"

	"github.com/basewarphq/bw/cmd/internal/cdkctx"
	"github.com/basewarphq/bw/cmd/internal/cmdexec"
	"github.com/basewarphq/bw/cmd/internal/wscfg"
)

type DiffCmd struct {
	Deployment string `arg:"" optional:"" help:"Deployment name (e.g., Stag, Prod). Defaults to claimed dev slot."`
}

func (c *DiffCmd) Run(cfg *wscfg.Config) error {
	ctx := context.Background()

	deployment, err := resolveDeployment(ctx, cfg, c.Deployment)
	if err != nil {
		return err
	}

	cctx, err := cdkctx.Load(cfg.CdkDir())
	if err != nil {
		return err
	}

	cdkArgs := cfg.Cdk.CdkArgs(cctx.Qualifier)
	args := make([]string, 0, 3+len(cdkArgs))
	args = append(args, "diff")
	args = append(args, cdkArgs...)
	args = append(args, cctx.Qualifier+"*Shared", cctx.Qualifier+"*"+deployment)
	return cmdexec.Run(ctx, cfg.CdkDir(), "cdk", args...)
}
