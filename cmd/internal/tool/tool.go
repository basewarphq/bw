package tool

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/cockroachdb/errors"
)

type Tool interface {
	Name() string
	RunsAfter() []string
}

type BinaryRequirement struct {
	Name   string
	Reason string
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

type Formatter interface {
	Fmt(ctx context.Context, dir string) error
}

type Generator interface {
	Gen(ctx context.Context, dir string) error
}

type Linter interface {
	Lint(ctx context.Context, dir string) error
}

type Compiler interface {
	Compiles(ctx context.Context, dir string) error
}

type Tester interface {
	UnitTest(ctx context.Context, dir string) error
}

func RunStep(ctx context.Context, target Tool, step Step, dir string) error {
	switch step {
	case StepFmt:
		if fmtr, ok := target.(Formatter); ok {
			return fmtr.Fmt(ctx, dir)
		}
	case StepGen:
		if gen, ok := target.(Generator); ok {
			return gen.Gen(ctx, dir)
		}
	case StepLint:
		if lntr, ok := target.(Linter); ok {
			return lntr.Lint(ctx, dir)
		}
	case StepCompiles:
		if comp, ok := target.(Compiler); ok {
			return comp.Compiles(ctx, dir)
		}
	case StepUnitTest:
		if tstr, ok := target.(Tester); ok {
			return tstr.UnitTest(ctx, dir)
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

func SupportsStep(t Tool, step Step) bool {
	switch step {
	case StepFmt:
		_, ok := t.(Formatter)
		return ok
	case StepGen:
		_, ok := t.(Generator)
		return ok
	case StepLint:
		_, ok := t.(Linter)
		return ok
	case StepCompiles:
		_, ok := t.(Compiler)
		return ok
	case StepUnitTest:
		_, ok := t.(Tester)
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
