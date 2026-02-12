package main

import (
	"github.com/basewarphq/bw/cmd/internal/tool/cdktool"
	"github.com/basewarphq/bw/cmd/internal/wscfg"
)

type InfraSlotsCmd struct {
	Claim   InfraSlotClaimCmd   `cmd:"" help:"Claim a free dev deployment slot."`
	Release InfraSlotReleaseCmd `cmd:"" help:"Release a claimed dev slot."`
	Status  InfraSlotStatusCmd  `cmd:"" help:"Show status of all dev slots."`
}

func infraProjectDirAndProfile(cfg *wscfg.Config) (dir, profile string, err error) {
	proj, err := cfg.FindProjectByTool("cdk")
	if err != nil {
		return "", "", err
	}
	dir = cfg.ProjectDir(*proj)
	if tc := cfg.ProjectToolConfig(proj.Name, "cdk"); tc != nil {
		profile = cdktool.ProfileFromConfig(tc)
	}
	return dir, profile, nil
}
