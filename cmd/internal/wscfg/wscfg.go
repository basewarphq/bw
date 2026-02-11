package wscfg

import (
	"os"
	"path/filepath"
	"slices"

	"github.com/BurntSushi/toml"
	"github.com/basewarphq/bw/cmd/internal/tool"
	"github.com/cockroachdb/errors"
)

const configFile = "bw.toml"

type Config struct {
	Root               string                    `toml:"-"`
	ProjectFilter      string                    `toml:"-"`
	NoDeps             bool                      `toml:"-"`
	Cli                []CliConfig               `toml:"cli"`
	Projects           []ProjectConfig           `toml:"project"`
	DecodedToolConfigs map[string]map[string]any `toml:"-"`
}

func (c *Config) FilteredProjects() []ProjectConfig {
	return FilterProjects(c.Projects, c.ProjectFilter, c.NoDeps)
}

type ProjectConfig struct {
	Name       string                    `toml:"name"`
	Dir        string                    `toml:"dir"`
	Tools      []string                  `toml:"tools"`
	DependsOn  []string                  `toml:"depends_on"`
	ToolConfig map[string]toml.Primitive `toml:"tool"`
}

type CliConfig struct {
	Name string `toml:"name"`
	Main string `toml:"main"`
}

func (c *Config) ProjectDir(proj ProjectConfig) string {
	return filepath.Join(c.Root, proj.Dir)
}

func (c *Config) FindProjectByTool(toolName string) (*ProjectConfig, error) {
	for i := range c.Projects {
		if slices.Contains(c.Projects[i].Tools, toolName) {
			return &c.Projects[i], nil
		}
	}
	return nil, errors.Newf("no project with tool %q found in workspace", toolName)
}

func Load(reg *tool.Registry) (*Config, error) {
	root, err := findRoot()
	if err != nil {
		return nil, err
	}

	var cfg Config
	meta, err := toml.DecodeFile(filepath.Join(root, configFile), &cfg)
	if err != nil {
		return nil, errors.Wrapf(err, "parsing %s", configFile)
	}

	cfg.Root = root

	if err := cfg.validate(); err != nil {
		return nil, errors.Wrapf(err, "invalid %s", configFile)
	}

	if err := cfg.decodeToolConfigs(meta, reg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) ProjectToolConfig(project, toolName string) any {
	if c.DecodedToolConfigs == nil {
		return nil
	}
	m, ok := c.DecodedToolConfigs[project]
	if !ok {
		return nil
	}
	return m[toolName]
}

func (c *Config) validate() error {
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

func FilterProjects(projects []ProjectConfig, name string, noDeps bool) []ProjectConfig {
	if name == "" {
		return projects
	}

	byName := make(map[string]ProjectConfig, len(projects))
	for _, p := range projects {
		byName[p.Name] = p
	}

	if _, ok := byName[name]; !ok {
		return projects
	}

	if noDeps {
		return []ProjectConfig{byName[name]}
	}

	visited := make(map[string]bool)
	var order []ProjectConfig

	var visit func(n string)
	visit = func(n string) {
		if visited[n] {
			return
		}
		visited[n] = true
		p, ok := byName[n]
		if !ok {
			return
		}
		for _, dep := range p.DependsOn {
			visit(dep)
		}
		order = append(order, p)
	}

	visit(name)
	return order
}

func (c *Config) decodeToolConfigs(meta toml.MetaData, reg *tool.Registry) error {
	c.DecodedToolConfigs = make(map[string]map[string]any)
	for _, proj := range c.Projects {
		if len(proj.ToolConfig) == 0 {
			continue
		}
		decoded := make(map[string]any, len(proj.ToolConfig))
		for toolName, raw := range proj.ToolConfig {
			tl, err := reg.Get(toolName)
			if err != nil {
				return errors.Wrapf(err, "project %q", proj.Name)
			}
			ct, ok := tl.(tool.Configurable)
			if !ok {
				return errors.Newf("project %q: tool %q does not accept configuration", proj.Name, toolName)
			}
			cfg, err := ct.DecodeConfig(meta, raw)
			if err != nil {
				return errors.Wrapf(err, "project %q: tool %q", proj.Name, toolName)
			}
			decoded[toolName] = cfg
		}
		c.DecodedToolConfigs[proj.Name] = decoded
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
