package main

import (
	"github.com/basewarphq/bw/cmd/internal/tool"
	"github.com/basewarphq/bw/cmd/internal/wscfg"
)

type DeployCmd struct {
	Deployment string `arg:"" optional:"" help:"Deployment name (e.g., Stag, Prod). Defaults to claimed dev slot."`
	Hotswap    bool   `help:"Enable CDK hotswap deployment for faster iterations."`
}

func (c *DeployCmd) Run(cfg *wscfg.Config, reg *tool.Registry) error {
	return (&InfraDeployCmd{
		Deployment: c.Deployment,
		Hotswap:    c.Hotswap,
	}).Run(cfg, reg)
}
