package tool

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/basewarphq/bw/cmd/internal/bincheck"
	"github.com/cockroachdb/errors"
)

type Tool interface {
	Name() string
	RunsAfter() []string
}

type BinaryRequirement struct {
	Name          string
	Reason        string
	SkipMiseCheck bool
}

type FileRequirement struct {
	Path   string
	Reason string
	Check  func(r io.Reader) error
}

type Doctor interface {
	RequiredBinaries() []BinaryRequirement
	RequiredFiles() []FileRequirement
}

type Configurable interface {
	DecodeConfig(meta toml.MetaData, raw toml.Primitive) (any, error)
}

type NodeReporter interface {
	Section(heading string)
	Table(columns []string, rows [][]string)
	Error(msg string)
}

type Reporter interface {
	ForNode(project, step, tool string) NodeReporter
}

type Initializer interface {
	Init(ctx context.Context, dir string, r NodeReporter) error
}

type Formatter interface {
	Fmt(ctx context.Context, dir string, r NodeReporter) error
}

type Generator interface {
	Gen(ctx context.Context, dir string, r NodeReporter) error
}

type Linter interface {
	Lint(ctx context.Context, dir string, r NodeReporter) error
}

type Builder interface {
	Build(ctx context.Context, dir string, r NodeReporter) error
}

type Tester interface {
	UnitTest(ctx context.Context, dir string, r NodeReporter) error
}

type Releaser interface {
	Release(ctx context.Context, dir string, r NodeReporter) error
}

type Bootstrapper interface {
	Bootstrap(ctx context.Context, dir string, r NodeReporter) error
}

type Differ interface {
	Diff(ctx context.Context, dir string, r NodeReporter) error
}

type Deployer interface {
	Deploy(ctx context.Context, dir string, r NodeReporter) error
}

func RunStep(ctx context.Context, target Tool, step Step, dir string, r NodeReporter) error {
	switch step {
	case StepInit:
		if init, ok := target.(Initializer); ok {
			return init.Init(ctx, dir, r)
		}
	case StepDoctor:
		if diag, ok := target.(Diagnoser); ok {
			return diag.Diagnose(ctx, dir, r)
		}
	case StepFmt:
		if fmtr, ok := target.(Formatter); ok {
			return fmtr.Fmt(ctx, dir, r)
		}
	case StepGen:
		if gen, ok := target.(Generator); ok {
			return gen.Gen(ctx, dir, r)
		}
	case StepLint:
		if lntr, ok := target.(Linter); ok {
			return lntr.Lint(ctx, dir, r)
		}
	case StepBuild:
		if b, ok := target.(Builder); ok {
			return b.Build(ctx, dir, r)
		}
	case StepUnitTest:
		if tstr, ok := target.(Tester); ok {
			return tstr.UnitTest(ctx, dir, r)
		}
	case StepRelease:
		if rel, ok := target.(Releaser); ok {
			return rel.Release(ctx, dir, r)
		}
	case StepBootstrap:
		if b, ok := target.(Bootstrapper); ok {
			return b.Bootstrap(ctx, dir, r)
		}
	case StepDiff:
		if d, ok := target.(Differ); ok {
			return d.Diff(ctx, dir, r)
		}
	case StepDeploy:
		if d, ok := target.(Deployer); ok {
			return d.Deploy(ctx, dir, r)
		}
	case StepInspect:
		if p, ok := target.(InspectionProvider); ok {
			return RunInspections(ctx, p, dir, r)
		}
	default:
		return errors.Newf("unknown step: %s", step)
	}
	return nil
}

type Registry struct {
	tools map[string]Tool
	order []string
}

func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

func (r *Registry) Register(t Tool) {
	r.tools[t.Name()] = t
	r.order = append(r.order, t.Name())
}

func (r *Registry) Get(name string) (Tool, error) {
	t, ok := r.tools[name]
	if !ok {
		return nil, errors.Newf("unknown tool: %q", name)
	}
	return t, nil
}

func (r *Registry) All() []Tool {
	result := make([]Tool, 0, len(r.order))
	for _, name := range r.order {
		result = append(result, r.tools[name])
	}
	return result
}

func SupportsStep(target Tool, step Step) bool {
	switch step {
	case StepInit:
		_, ok := target.(Initializer)
		return ok
	case StepDoctor:
		_, ok := target.(Diagnoser)
		return ok
	case StepFmt:
		_, ok := target.(Formatter)
		return ok
	case StepGen:
		_, ok := target.(Generator)
		return ok
	case StepLint:
		_, ok := target.(Linter)
		return ok
	case StepBuild:
		_, ok := target.(Builder)
		return ok
	case StepUnitTest:
		_, ok := target.(Tester)
		return ok
	case StepRelease:
		_, ok := target.(Releaser)
		return ok
	case StepBootstrap:
		_, ok := target.(Bootstrapper)
		return ok
	case StepDiff:
		_, ok := target.(Differ)
		return ok
	case StepDeploy:
		_, ok := target.(Deployer)
		return ok
	case StepInspect:
		_, ok := target.(InspectionProvider)
		return ok
	default:
		return false
	}
}

func CheckFiles(dir string, reqs []FileRequirement) error {
	for _, req := range reqs {
		fullPath := filepath.Join(dir, req.Path)

		if req.Check != nil {
			fl, err := os.Open(fullPath)
			if err != nil {
				return errors.Newf("required file %q not found in %s (%s)", req.Path, dir, req.Reason)
			}

			checkErr := req.Check(fl)
			fl.Close()

			if checkErr != nil {
				return errors.Wrapf(checkErr, "file %q in %s", req.Path, dir)
			}
		} else {
			if _, err := os.Stat(fullPath); err != nil {
				return errors.Newf("required file %q not found in %s (%s)", req.Path, dir, req.Reason)
			}
		}
	}

	return nil
}

type nopNodeReporter struct{}

func (nopNodeReporter) Section(string)             {}
func (nopNodeReporter) Table([]string, [][]string) {}
func (nopNodeReporter) Error(string)               {}

type nopReporter struct{}

func (nopReporter) ForNode(string, string, string) NodeReporter { return nopNodeReporter{} }

func NopReporter() Reporter { return nopReporter{} }

type toolConfigKey struct{}

func WithToolConfig(ctx context.Context, cfg any) context.Context {
	return context.WithValue(ctx, toolConfigKey{}, cfg)
}

func ToolConfigFrom[T any](ctx context.Context) *T {
	v, ok := ctx.Value(toolConfigKey{}).(T)
	if !ok {
		return nil
	}
	return &v
}

type deploymentKey struct{}

func WithDeployment(ctx context.Context, d string) context.Context {
	return context.WithValue(ctx, deploymentKey{}, d)
}

func DeploymentFrom(ctx context.Context) (string, bool) {
	d, ok := ctx.Value(deploymentKey{}).(string)
	return d, ok
}

type BootstrapOptions struct {
	Profile             string
	ExecutionPolicies   string
	PermissionsBoundary string
}

type bootstrapOptionsKey struct{}

func WithBootstrapOptions(ctx context.Context, opts BootstrapOptions) context.Context {
	return context.WithValue(ctx, bootstrapOptionsKey{}, opts)
}

func BootstrapOptionsFrom(ctx context.Context) (BootstrapOptions, bool) {
	opts, ok := ctx.Value(bootstrapOptionsKey{}).(BootstrapOptions)
	return opts, ok
}

type DeployOptions struct {
	Hotswap bool
}

type binCheckerKey struct{}

func WithBinChecker(ctx context.Context, bc *bincheck.Checker) context.Context {
	return context.WithValue(ctx, binCheckerKey{}, bc)
}

func BinCheckerFrom(ctx context.Context) *bincheck.Checker {
	bc, ok := ctx.Value(binCheckerKey{}).(*bincheck.Checker)
	if !ok {
		return bincheck.NewChecker()
	}
	return bc
}

type deployOptionsKey struct{}

func WithDeployOptions(ctx context.Context, opts DeployOptions) context.Context {
	return context.WithValue(ctx, deployOptionsKey{}, opts)
}

func DeployOptionsFrom(ctx context.Context) (DeployOptions, bool) {
	opts, ok := ctx.Value(deployOptionsKey{}).(DeployOptions)
	return opts, ok
}

type ReleaseOptions struct {
	DryRun bool
}

type releaseOptionsKey struct{}

func WithReleaseOptions(ctx context.Context, opts ReleaseOptions) context.Context {
	return context.WithValue(ctx, releaseOptionsKey{}, opts)
}

func ReleaseOptionsFrom(ctx context.Context) (ReleaseOptions, bool) {
	opts, ok := ctx.Value(releaseOptionsKey{}).(ReleaseOptions)
	return opts, ok
}
