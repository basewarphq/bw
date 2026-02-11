package openapitool_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/basewarphq/bw/cmd/internal/testutil"
	"github.com/basewarphq/bw/cmd/internal/tool/openapitool"
)

const minimalOpenAPI31 = `{
  "openapi": "3.1.0",
  "info": {
    "title": "Test API",
    "version": "1.0.0"
  },
  "paths": {}
}`

func TestGenDownconverts(t *testing.T) {
	t.Parallel()
	testutil.RequireBinary(t, "openapi-down-convert")

	dir := testutil.Setup(t, map[string]string{
		"proto/openapi.json": minimalOpenAPI31,
	})

	tl := openapitool.New()
	if err := tl.Gen(context.Background(), dir, nil); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "proto/openapi-3.0.json"))
	if err != nil {
		t.Fatalf("expected proto/openapi-3.0.json to exist: %v", err)
	}

	if !strings.Contains(string(got), "3.0") {
		t.Error("expected output to contain \"3.0\"")
	}
}

func TestGenMissingInputErrors(t *testing.T) {
	t.Parallel()
	testutil.RequireBinary(t, "openapi-down-convert")

	dir := testutil.Setup(t, map[string]string{})

	tl := openapitool.New()
	if err := tl.Gen(context.Background(), dir, nil); err == nil {
		t.Error("expected error when proto/openapi.json is missing")
	}
}
