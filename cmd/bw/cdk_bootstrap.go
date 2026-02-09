package main

import (
	"context"

	"github.com/basewarphq/bw/cmd/internal/cmdexec"
	"github.com/basewarphq/bw/cmd/internal/projcfg"
)

type BootstrapCmd struct{}

func (c *BootstrapCmd) Run(cfg *projcfg.Config) error {
	return cmdexec.Run(context.Background(), cfg.CdkDir(), "cdk", "bootstrap")
}
