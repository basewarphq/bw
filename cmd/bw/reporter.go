package main

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/basewarphq/bw/cmd/internal/tool"
)

type cliReporter struct{}

func (cliReporter) ForNode(_, _, _ string) tool.NodeReporter {
	return &cliNodeReporter{}
}

type cliNodeReporter struct{}

func (r *cliNodeReporter) Section(heading string) {
	fmt.Fprintf(os.Stdout, "=== %s ===\n", heading)
}

func (r *cliNodeReporter) Table(columns []string, rows [][]string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, strings.Join(columns, "\t"))
	for _, row := range rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	w.Flush()
}

func (r *cliNodeReporter) Error(msg string) {
	fmt.Fprintln(os.Stderr, msg)
}
