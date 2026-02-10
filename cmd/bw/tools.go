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
	steps := tool.AllDevCheckSteps

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

	return writer.Flush()
}
