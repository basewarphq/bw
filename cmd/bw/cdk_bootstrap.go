package main

import (
	"context"

	"github.com/basewarphq/bw/cmd/internal/cdkctx"
	"github.com/basewarphq/bw/cmd/internal/cmdexec"
	"github.com/basewarphq/bw/cmd/internal/projcfg"
)

type BootstrapCmd struct{}

func (c *BootstrapCmd) Run(cfg *projcfg.Config) error {
	cctx, err := cdkctx.Load(cfg.CdkDir())
	if err != nil {
		return err
	}

	cdkArgs := cfg.Cdk.CdkArgs(cctx.Qualifier)
	args := make([]string, 0, 1+len(cdkArgs))
	args = append(args, "bootstrap")
	args = append(args, cdkArgs...)
	return cmdexec.Run(context.Background(), cfg.CdkDir(), "cdk", args...)
}
