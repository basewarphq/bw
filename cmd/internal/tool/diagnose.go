package tool

import (
	"context"
	"fmt"
	"strings"

	"github.com/basewarphq/bw/cmd/internal/bincheck"
	"github.com/cockroachdb/errors"
)

type Diagnoser interface {
	Diagnose(ctx context.Context, dir string, r NodeReporter) error
}

func DiagnoseDefaults(ctx context.Context, dir string, doc Doctor, bc *bincheck.Checker, r NodeReporter) error {
	var errs []string

	for _, bin := range doc.RequiredBinaries() {
		res := bc.Check(ctx, bin.Name)
		switch {
		case res.MiseManaged && res.InPath:
			r.Table(nil, [][]string{{"✓", bin.Name, "(mise)"}})
		case bin.SkipMiseCheck && res.InPath:
			r.Table(nil, [][]string{{"✓", bin.Name, "(system)"}})
		case !res.MiseManaged && res.InPath:
			msg := fmt.Sprintf("%s found in PATH but not managed by mise", bin.Name)
			r.Error("✗ " + msg)
			errs = append(errs, msg)
		default:
			msg := fmt.Sprintf("%s not found (%s)", bin.Name, bin.Reason)
			r.Error("✗ " + msg)
			errs = append(errs, msg)
		}
	}

	if err := CheckFiles(dir, doc.RequiredFiles()); err != nil {
		r.Error(err.Error())
		errs = append(errs, err.Error())
	} else {
		for _, req := range doc.RequiredFiles() {
			r.Table(nil, [][]string{{"✓", req.Path}})
		}
	}

	if len(errs) > 0 {
		return errors.Newf("doctor checks failed: %s", strings.Join(errs, "; "))
	}
	return nil
}
