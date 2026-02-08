package cmdexec

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cockroachdb/errors"
)

type Error struct {
	Cmd      string
	Args     []string
	Dir      string
	ExitCode int
	Stderr   string
}

func (e *Error) Error() string {
	msg := fmt.Sprintf("(in %s) %s %s", e.Dir, e.Cmd, strings.Join(e.Args, " "))
	if e.Stderr != "" {
		return fmt.Sprintf("%s: exit %d\n%s", msg, e.ExitCode, strings.TrimSpace(e.Stderr))
	}
	return fmt.Sprintf("%s: exit %d", msg, e.ExitCode)
}

func Output(ctx context.Context, dir, name string, args ...string) (string, error) {
	if !filepath.IsAbs(dir) {
		return "", errors.Newf("cmdexec: dir must be absolute, got %q", dir)
	}

	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	if err != nil {
		return "", wrapErr(dir, name, args, err, stderr.String())
	}
	return string(out), nil
}

func Run(ctx context.Context, dir, name string, args ...string) error {
	if !filepath.IsAbs(dir) {
		return errors.Newf("cmdexec: dir must be absolute, got %q", dir)
	}

	var stderrBuf bytes.Buffer
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)

	if err := cmd.Run(); err != nil {
		return wrapErr(dir, name, args, err, stderrBuf.String())
	}
	return nil
}

func wrapErr(dir, name string, args []string, err error, stderr string) error {
	exitCode := 1
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		exitCode = exitErr.ExitCode()
		if stderr == "" {
			stderr = string(exitErr.Stderr)
		}
	}
	return &Error{
		Cmd:      name,
		Args:     args,
		Dir:      dir,
		ExitCode: exitCode,
		Stderr:   stderr,
	}
}
