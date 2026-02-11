package main

import "github.com/basewarphq/bw/cmd/internal/wscfg"

type SlotClaimCmd struct{}

func (c *SlotClaimCmd) Run(cfg *wscfg.Config) error {
	return (&InfraSlotClaimCmd{}).Run(cfg)
}
