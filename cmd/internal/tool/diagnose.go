package tool

import (
	"context"
	"fmt"

	"github.com/basewarphq/bw/cmd/internal/bincheck"
	"github.com/cockroachdb/errors"
)

type Diagnoser interface {
	Diagnose(ctx context.Context, dir string, r NodeReporter) error
}

func DiagnoseDefaults(ctx context.Context, dir string, doc Doctor, bc *bincheck.Checker, r NodeReporter) error {
	failed := false

	for _, bin := range doc.RequiredBinaries() {
		res := bc.Check(ctx, bin.Name)
		switch {
		case res.MiseManaged && res.InPath:
			r.Table(nil, [][]string{{"✓", bin.Name, "(mise)"}})
		case bin.SkipMiseCheck && res.InPath:
			r.Table(nil, [][]string{{"✓", bin.Name, "(system)"}})
		case !res.MiseManaged && res.InPath:
			r.Error(fmt.Sprintf("✗ %s found in PATH but not managed by mise", bin.Name))
			failed = true
		default:
			r.Error(fmt.Sprintf("✗ %s not found (%s)", bin.Name, bin.Reason))
			failed = true
		}
	}

	if err := CheckFiles(dir, doc.RequiredFiles()); err != nil {
		r.Error(err.Error())
		failed = true
	} else {
		for _, req := range doc.RequiredFiles() {
			r.Table(nil, [][]string{{"✓", req.Path}})
		}
	}

	if failed {
		return errors.New("doctor checks failed")
	}
	return nil
}
