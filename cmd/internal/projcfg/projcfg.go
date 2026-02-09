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
	Dir string `toml:"dir"`
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
