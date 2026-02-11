package main

import "github.com/basewarphq/bw/cmd/internal/wscfg"

type SlotReleaseCmd struct {
	Slot  string `help:"Name of the slot to release (default: this checkout's claimed slot)." short:"s"`
	Force bool   `help:"Force-release the slot even if it belongs to someone else." short:"f"`
}

func (c *SlotReleaseCmd) Run(cfg *wscfg.Config) error {
	return (&InfraSlotReleaseCmd{
		Slot:  c.Slot,
		Force: c.Force,
	}).Run(cfg)
}
