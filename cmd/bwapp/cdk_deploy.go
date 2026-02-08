package main

import (
	"context"

	"github.com/basewarphq/bwapp/cmd/internal/cmdexec"
	"github.com/basewarphq/bwapp/cmd/internal/projcfg"
)

type DeployCmd struct {
	Deployment string `arg:"" required:"" help:"Deployment name (e.g., Staging, Prod)."`
	Hotswap    bool   `help:"Enable CDK hotswap deployment for faster iterations."`
}

func (c *DeployCmd) Run(cfg *projcfg.Config) error {
	args := []string{"deploy", "--require-approval", "never"}
	if c.Hotswap {
		args = append(args, "--hotswap")
	}
	args = append(args, "bwapp*"+c.Deployment)
	return cmdexec.Run(context.Background(), cfg.CdkDir(), "cdk", args...)
}
