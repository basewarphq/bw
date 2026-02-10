package wscfg

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/cockroachdb/errors"
)

const configFile = "bw.toml"

type Config struct {
	Root     string          `toml:"-"`
	Cdk      CdkConfig       `toml:"cdk"`
	Cli      []CliConfig     `toml:"cli"`
	Projects []ProjectConfig `toml:"project"`
}

type ProjectConfig struct {
	Name      string   `toml:"name"`
	Dir       string   `toml:"dir"`
	Tools     []string `toml:"tools"`
	DependsOn []string `toml:"depends_on"`
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

func (c *Config) ProjectDir(proj ProjectConfig) string {
	return filepath.Join(c.Root, proj.Dir)
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
	if err := c.Cdk.validate(); err != nil {
		return err
	}
	for i, cli := range c.Cli {
		if cli.Name == "" {
			return errors.Newf("cli[%d].name is required", i)
		}
		if cli.Main == "" {
			return errors.Newf("cli[%d].main is required", i)
		}
	}
	return validateProjects(c.Projects)
}

func (c *CdkConfig) validate() error {
	if c.Dir == "" {
		return errors.New("cdk.dir is required")
	}
	if filepath.IsAbs(c.Dir) {
		return errors.Newf("cdk.dir must be relative, got %q", c.Dir)
	}
	if c.DevStrategy != "" && c.DevStrategy != "iam-username" {
		return errors.Newf("cdk.dev-strategy must be %q, got %q", "iam-username", c.DevStrategy)
	}
	if pb := c.PreBootstrap; pb != nil {
		if pb.Template == "" {
			return errors.New("cdk.pre-bootstrap.template is required")
		}
		if filepath.IsAbs(pb.Template) {
			return errors.Newf("cdk.pre-bootstrap.template must be relative, got %q", pb.Template)
		}
	}
	return nil
}

func validateProjects(projects []ProjectConfig) error {
	names := make(map[string]struct{}, len(projects))
	for i, proj := range projects {
		if proj.Name == "" {
			return errors.Newf("project[%d].name is required", i)
		}
		if proj.Dir == "" {
			return errors.Newf("project[%d].dir is required", i)
		}
		if filepath.IsAbs(proj.Dir) {
			return errors.Newf("project[%d].dir must be relative, got %q", i, proj.Dir)
		}
		if len(proj.Tools) == 0 {
			return errors.Newf("project[%d].tools is required", i)
		}
		if _, dup := names[proj.Name]; dup {
			return errors.Newf("duplicate project name %q", proj.Name)
		}
		names[proj.Name] = struct{}{}
	}
	for i, proj := range projects {
		for _, dep := range proj.DependsOn {
			if _, ok := names[dep]; !ok {
				return errors.Newf("project[%d] (%q) depends on unknown project %q", i, proj.Name, dep)
			}
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
