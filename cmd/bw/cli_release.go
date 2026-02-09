package main

import (
	"context"
	"os"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/basewarphq/bw/cmd/internal/cmdexec"
	"github.com/basewarphq/bw/cmd/internal/projcfg"
	"github.com/cockroachdb/errors"
)

type CliReleaseCmd struct {
	Minor bool `help:"Bump minor version instead of patch."`
}

func (c *CliReleaseCmd) Run(cfg *projcfg.Config) error {
	ctx := context.Background()

	if _, err := cmdexec.Output(ctx, cfg.Root, "git", "diff", "--quiet", "HEAD"); err != nil {
		return errors.New("git worktree is dirty, commit or stash changes first")
	}

	tag, err := latestTag(ctx, cfg.Root)
	if err != nil {
		return err
	}

	cur, err := semver.NewVersion(tag)
	if err != nil {
		return errors.Wrapf(err, "parsing tag %q", tag)
	}

	var next semver.Version
	if c.Minor {
		next = cur.IncMinor()
	} else {
		next = cur.IncPatch()
	}
	nextTag := "v" + next.String()

	_, _ = os.Stderr.WriteString("releasing " + tag + " -> " + nextTag + "\n")

	if err := cmdexec.Run(ctx, cfg.Root, "git", "tag", "-a", nextTag, "-m", "Release "+nextTag); err != nil {
		return errors.Wrap(err, "creating tag")
	}
	if err := cmdexec.Run(ctx, cfg.Root, "git", "push", "origin", nextTag); err != nil {
		return errors.Wrap(err, "pushing tag")
	}

	return cmdexec.Run(ctx, cfg.Root, "goreleaser", "release", "--clean")
}

func latestTag(ctx context.Context, dir string) (string, error) {
	out, err := cmdexec.Output(ctx, dir, "git", "describe", "--tags", "--abbrev=0", "--match", "v*")
	if err != nil {
		return "v0.0.0", nil //nolint:nilerr // no tags yet is not an error
	}
	return strings.TrimSpace(out), nil
}
