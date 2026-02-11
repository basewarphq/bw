package onepasswordtool

import (
	"context"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/basewarphq/bw/cmd/internal/cmdexec"
	"github.com/basewarphq/bw/cmd/internal/tool"
	"github.com/cockroachdb/errors"
)

type opConfig struct {
	EnvTemplate string `toml:"env-template"`
	EnvOutput   string `toml:"env-output"`
}

type Tool struct{}

func New() *Tool { return &Tool{} }

func (t *Tool) Name() string        { return "1password" }
func (t *Tool) RunsAfter() []string { return nil }

func (t *Tool) DecodeConfig(meta toml.MetaData, raw toml.Primitive) (any, error) {
	var cfg opConfig
	if err := meta.PrimitiveDecode(raw, &cfg); err != nil {
		return nil, errors.Wrap(err, "decoding 1password config")
	}
	if cfg.EnvTemplate == "" {
		return nil, errors.New("env-template is required")
	}
	if cfg.EnvOutput == "" {
		return nil, errors.New("env-output is required")
	}
	if filepath.IsAbs(cfg.EnvTemplate) {
		return nil, errors.Newf("env-template must be relative, got %q", cfg.EnvTemplate)
	}
	if filepath.IsAbs(cfg.EnvOutput) {
		return nil, errors.Newf("env-output must be relative, got %q", cfg.EnvOutput)
	}
	return cfg, nil
}

func (t *Tool) RequiredBinaries() []tool.BinaryRequirement {
	return []tool.BinaryRequirement{
		{Name: "op", Reason: "inject secrets from 1Password"},
	}
}

func (t *Tool) RequiredFiles() []tool.FileRequirement {
	return nil
}

func (t *Tool) Diagnose(ctx context.Context, dir string, r tool.NodeReporter) error {
	if err := tool.DiagnoseDefaults(ctx, dir, t, tool.BinCheckerFrom(ctx), r); err != nil {
		return err
	}
	if cfg := configFromCtx(ctx); cfg != nil {
		reqs := []tool.FileRequirement{{
			Path:   cfg.EnvTemplate,
			Reason: "1Password env template",
		}}
		if err := tool.CheckFiles(dir, reqs); err != nil {
			r.Error(err.Error())
			return err
		}
		r.Table(nil, [][]string{{"✓", cfg.EnvTemplate}})
	}
	return nil
}

func (t *Tool) Init(ctx context.Context, dir string, _ tool.NodeReporter) error {
	cfg := configFromCtx(ctx)
	if cfg == nil {
		return errors.New("1password tool requires configuration (env-template and env-output)")
	}
	return cmdexec.Run(ctx, dir, "op", "inject", "-i", cfg.EnvTemplate, "-o", cfg.EnvOutput, "-f")
}

func configFromCtx(ctx context.Context) *opConfig {
	return tool.ToolConfigFrom[opConfig](ctx)
}
