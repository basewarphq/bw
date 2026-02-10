package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"slices"

	"github.com/basewarphq/bw/cmd/internal/tool"
	"github.com/basewarphq/bw/cmd/internal/wscfg"
	"github.com/cockroachdb/errors"
)

type DoctorCmd struct{}

func (c *DoctorCmd) Run(cfg *wscfg.Config, reg *tool.Registry) error {
	toolsNeeded := collectToolsForProjects(cfg, reg)
	var failed bool

	for _, tl := range toolsNeeded {
		doc, ok := tl.(tool.Doctor)
		if !ok {
			continue
		}

		fmt.Fprintf(os.Stdout, "=== %s ===\n", tl.Name())

		failed = checkBinaries(doc) || failed
		failed = checkProjectFiles(cfg, tl, doc) || failed

		fmt.Fprintln(os.Stdout)
	}

	if failed {
		return errors.New("doctor found problems; see above")
	}

	fmt.Fprintln(os.Stdout, "All checks passed.")
	return nil
}

func collectToolsForProjects(
	cfg *wscfg.Config,
	reg *tool.Registry,
) []tool.Tool {
	seen := make(map[string]struct{})
	var result []tool.Tool

	for _, proj := range cfg.Projects {
		for _, toolName := range proj.Tools {
			if _, ok := seen[toolName]; ok {
				continue
			}
			seen[toolName] = struct{}{}

			tl, err := reg.Get(toolName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "  WARNING: unknown tool %q in project %q\n", toolName, proj.Name)
				continue
			}
			result = append(result, tl)
		}
	}
	return result
}

func checkBinaries(doc tool.Doctor) bool {
	var failed bool
	for _, bin := range doc.RequiredBinaries() {
		miseManaged := isMiseManaged(bin.Name)
		_, lookErr := exec.LookPath(bin.Name)
		inPath := lookErr == nil

		switch {
		case miseManaged && inPath:
			fmt.Fprintf(os.Stdout, "  ✓ %s (mise)\n", bin.Name)
		case !miseManaged && inPath:
			fmt.Fprintf(os.Stdout, "  ✗ %s found in PATH but not managed by mise\n", bin.Name)
			failed = true
		default:
			fmt.Fprintf(os.Stdout, "  ✗ %s not found (%s)\n", bin.Name, bin.Reason)
			failed = true
		}
	}
	return failed
}

func checkProjectFiles(
	cfg *wscfg.Config,
	tl tool.Tool,
	doc tool.Doctor,
) bool {
	reqs := doc.RequiredFiles()
	if len(reqs) == 0 {
		return false
	}

	var failed bool
	for _, proj := range cfg.Projects {
		if !slices.Contains(proj.Tools, tl.Name()) {
			continue
		}

		projDir := cfg.ProjectDir(proj)
		if err := tool.CheckFiles(projDir, reqs); err != nil {
			fmt.Fprintf(os.Stdout, "  ✗ %s: %v\n", proj.Name, err)
			failed = true
		} else {
			for _, req := range reqs {
				fmt.Fprintf(os.Stdout, "  ✓ %s: %s\n", proj.Name, req.Path)
			}
		}
	}
	return failed
}

func isMiseManaged(binary string) bool {
	cmd := exec.CommandContext(context.Background(), "mise", "which", binary)
	return cmd.Run() == nil
}
