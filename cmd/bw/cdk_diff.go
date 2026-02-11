package main

import (
	"github.com/basewarphq/bw/cmd/internal/tool"
	"github.com/basewarphq/bw/cmd/internal/wscfg"
)

type DiffCmd struct {
	Deployment string `arg:"" optional:"" help:"Deployment name (e.g., Stag, Prod). Defaults to claimed dev slot."`
}

func (c *DiffCmd) Run(cfg *wscfg.Config, reg *tool.Registry) error {
	return (&InfraDiffCmd{
		Deployment: c.Deployment,
	}).Run(cfg, reg)
}
