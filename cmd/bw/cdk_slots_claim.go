package main

import (
	"context"
	"fmt"
	"os"

	"github.com/basewarphq/bw/cmd/internal/wscfg"
)

type SlotClaimCmd struct{}

func (c *SlotClaimCmd) Run(cfg *wscfg.Config) error {
	claim, err := ensureClaim(context.Background(), cfg)
	if err != nil {
		return err
	}

	fmt.Fprintln(os.Stdout, claim.Slot)
	return nil
}
