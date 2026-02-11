package goreleasertool

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/Masterminds/semver/v3"
	"github.com/basewarphq/bw/cmd/internal/cmdexec"
	"github.com/basewarphq/bw/cmd/internal/tool"
	"github.com/cockroachdb/errors"
)

type goreleaserConfig struct {
	VersionFile string `toml:"version-file"`
}

type Tool struct{}

func New() *Tool { return &Tool{} }

func (t *Tool) Name() string        { return "goreleaser" }
func (t *Tool) RunsAfter() []string { return nil }

func (t *Tool) RequiredBinaries() []tool.BinaryRequirement {
	return []tool.BinaryRequirement{
		{Name: "goreleaser", Reason: "build and release Go binaries"},
		{Name: "git", Reason: "tag and push releases", SkipMiseCheck: true},
	}
}

func (t *Tool) RequiredFiles() []tool.FileRequirement {
	return []tool.FileRequirement{
		{Path: ".goreleaser.yaml", Reason: "GoReleaser configuration"},
	}
}

func (t *Tool) Diagnose(ctx context.Context, dir string, r tool.NodeReporter) error {
	return tool.DiagnoseDefaults(ctx, dir, t, tool.BinCheckerFrom(ctx), r)
}

func (t *Tool) DecodeConfig(meta toml.MetaData, raw toml.Primitive) (any, error) {
	var cfg goreleaserConfig
	if err := meta.PrimitiveDecode(raw, &cfg); err != nil {
		return nil, errors.Wrap(err, "decoding goreleaser config")
	}
	if cfg.VersionFile == "" {
		return nil, errors.New("version-file is required")
	}
	return cfg, nil
}

func (t *Tool) Build(ctx context.Context, dir string, _ tool.NodeReporter) error {
	return cmdexec.Run(ctx, dir, "goreleaser", "build", "--snapshot", "--clean")
}

func (t *Tool) Release(ctx context.Context, dir string, _ tool.NodeReporter) error {
	cfg := configFromCtx(ctx)

	raw, err := os.ReadFile(filepath.Join(dir, cfg.VersionFile))
	if err != nil {
		return errors.Wrap(err, "reading version file")
	}
	version := strings.TrimSpace(string(raw))

	if _, err := semver.StrictNewVersion(version); err != nil {
		return errors.Newf("invalid semver in %s: %q", cfg.VersionFile, version)
	}

	tag := "v" + version

	if err := cmdexec.Run(ctx, dir, "git", "diff", "--quiet", "HEAD"); err != nil {
		return errors.New("git worktree is dirty, commit or stash changes first")
	}

	out, err := cmdexec.Output(ctx, dir, "git", "tag", "-l", tag)
	if err != nil {
		return errors.Wrap(err, "checking existing tags")
	}
	if strings.TrimSpace(out) != "" {
		return errors.Newf("tag %s already exists", tag)
	}

	opts, _ := tool.ReleaseOptionsFrom(ctx)

	if err := cmdexec.Run(ctx, dir, "git", "tag", "-a", tag, "-m", "Release "+tag); err != nil {
		return errors.Wrap(err, "creating tag")
	}

	if opts.DryRun {
		return cmdexec.Run(ctx, dir, "goreleaser", "release", "--snapshot", "--clean")
	}

	if err := cmdexec.Run(ctx, dir, "git", "push", "origin", tag); err != nil {
		return errors.Wrap(err, "pushing tag")
	}

	return cmdexec.Run(ctx, dir, "goreleaser", "release", "--clean")
}

func configFromCtx(ctx context.Context) *goreleaserConfig {
	cfg := tool.ToolConfigFrom[goreleaserConfig](ctx)
	if cfg == nil {
		return &goreleaserConfig{}
	}
	return cfg
}
