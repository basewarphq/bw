package projcfg

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/cockroachdb/errors"
)

const configFile = "bw.toml"

type Config struct {
	Root string      `toml:"-"`
	Cdk  CdkConfig   `toml:"cdk"`
	Cli  []CliConfig `toml:"cli"`
}

type CdkConfig struct {
	Dir             string              `toml:"dir"`
	Profile         string              `toml:"profile"`
	DevStrategy     string              `toml:"dev-strategy"`
	LegacyBootstrap bool                `toml:"legacy-bootstrap"`
	PreBootstrap    *PreBootstrapConfig `toml:"pre-bootstrap"`
}

type PreBootstrapConfig struct {
	Template   string            `toml:"template"`
	Parameters map[string]string `toml:"parameters"`
}

func (c *CdkConfig) CdkArgs(qualifier string) []string {
	var args []string
	if c.LegacyBootstrap {
		args = append(args,
			"--qualifier", qualifier,
			"--toolkit-stack-name", qualifier+"Bootstrap",
		)
	}
	if c.Profile != "" {
		args = append(args, "--profile", c.Profile)
	}
	return args
}

func (c *CdkConfig) AwsArgs() []string {
	if c.Profile != "" {
		return []string{"--profile", c.Profile}
	}
	return nil
}

type CliConfig struct {
	Name string `toml:"name"`
	Main string `toml:"main"`
}

func (c *Config) CdkDir() string {
	return filepath.Join(c.Root, c.Cdk.Dir)
}

func Load() (*Config, error) {
	root, err := findRoot()
	if err != nil {
		return nil, err
	}

	var cfg Config
	if _, err := toml.DecodeFile(filepath.Join(root, configFile), &cfg); err != nil {
		return nil, errors.Wrapf(err, "parsing %s", configFile)
	}

	cfg.Root = root

	if err := cfg.validate(); err != nil {
		return nil, errors.Wrapf(err, "invalid %s", configFile)
	}

	return &cfg, nil
}

func (c *Config) validate() error {
	if c.Cdk.Dir == "" {
		return errors.New("cdk.dir is required")
	}
	if filepath.IsAbs(c.Cdk.Dir) {
		return errors.Newf("cdk.dir must be relative, got %q", c.Cdk.Dir)
	}
	if c.Cdk.DevStrategy != "" && c.Cdk.DevStrategy != "iam-username" {
		return errors.Newf("cdk.dev-strategy must be %q, got %q", "iam-username", c.Cdk.DevStrategy)
	}
	if pb := c.Cdk.PreBootstrap; pb != nil {
		if pb.Template == "" {
			return errors.New("cdk.pre-bootstrap.template is required")
		}
		if filepath.IsAbs(pb.Template) {
			return errors.Newf("cdk.pre-bootstrap.template must be relative, got %q", pb.Template)
		}
	}
	for i, cli := range c.Cli {
		if cli.Name == "" {
			return errors.Newf("cli[%d].name is required", i)
		}
		if cli.Main == "" {
			return errors.Newf("cli[%d].main is required", i)
		}
	}
	return nil
}

func findRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, configFile)); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.Newf("could not find %s in any parent directory", configFile)
		}
		dir = parent
	}
}
