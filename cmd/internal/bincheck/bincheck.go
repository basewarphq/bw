package bincheck

import (
	"context"
	"os/exec"
	"sync"
)

type Result struct {
	InPath      bool
	MiseManaged bool
}

type Checker struct {
	cache sync.Map
}

func NewChecker() *Checker {
	return &Checker{}
}

func (c *Checker) Check(ctx context.Context, name string) Result {
	if v, ok := c.cache.Load(name); ok {
		r, _ := v.(Result)
		return r
	}

	r := Result{
		InPath:      lookPath(name),
		MiseManaged: isMiseManaged(ctx, name),
	}

	actual, _ := c.cache.LoadOrStore(name, r)
	stored, _ := actual.(Result)
	return stored
}

func lookPath(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func isMiseManaged(ctx context.Context, binary string) bool {
	cmd := exec.CommandContext(ctx, "mise", "which", binary)
	return cmd.Run() == nil
}
