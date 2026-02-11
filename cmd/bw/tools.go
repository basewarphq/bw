package main

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/basewarphq/bw/cmd/internal/tool"
)

type ToolsMatrixCmd struct{}

func (c *ToolsMatrixCmd) Run(reg *tool.Registry) error {
	allTools := reg.All()
	steps := tool.AllSteps

	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	var header strings.Builder
	header.WriteString("TOOL")
	for _, step := range steps {
		header.WriteString("\t")
		header.WriteString(strings.ToUpper(step.String()))
	}
	fmt.Fprintln(writer, header.String())

	for _, tl := range allTools {
		var row strings.Builder
		row.WriteString(tl.Name())
		for _, step := range steps {
			if tool.SupportsStep(tl, step) {
				row.WriteString("\t✓")
			} else {
				row.WriteString("\t-")
			}
		}
		fmt.Fprintln(writer, row.String())
	}

	if err := writer.Flush(); err != nil {
		return err
	}

	var hasLenses bool
	for _, tl := range allTools {
		names := tool.InspectionNames(tl)
		if len(names) > 0 {
			if !hasLenses {
				fmt.Fprintln(os.Stdout)
				fmt.Fprintln(os.Stdout, "Inspect lenses:")
				hasLenses = true
			}
			fmt.Fprintf(os.Stdout, "  %s: %s\n", tl.Name(), strings.Join(names, ", "))
		}
	}

	return nil
}
