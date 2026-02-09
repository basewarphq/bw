package main

import (
	"context"
	"fmt"
	"os"

	"github.com/basewarphq/bw/cmd/internal/projcfg"
)

type SlotClaimCmd struct{}

func (c *SlotClaimCmd) Run(cfg *projcfg.Config) error {
	claim, err := ensureClaim(context.Background(), cfg)
	if err != nil {
		return err
	}

	fmt.Fprintln(os.Stdout, claim.Slot)
	return nil
}
