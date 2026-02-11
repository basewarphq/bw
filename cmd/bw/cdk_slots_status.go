package main

import "github.com/basewarphq/bw/cmd/internal/wscfg"

type SlotStatusCmd struct{}

func (c *SlotStatusCmd) Run(cfg *wscfg.Config) error {
	return (&InfraSlotStatusCmd{}).Run(cfg)
}
