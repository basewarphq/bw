package cfnparams_test

import (
	"strings"
	"testing"

	"github.com/basewarphq/bw/cmd/internal/cfnparams"
)

func TestResolve_StaticValues(t *testing.T) {
	t.Parallel()
	raw := map[string]string{"Repo": "basewarphq/bw"}
	got, err := cfnparams.Resolve(raw, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got["Repo"] != "basewarphq/bw" {
		t.Errorf("got %q, want %q", got["Repo"], "basewarphq/bw")
	}
}

func TestResolve_SinglePlaceholder(t *testing.T) {
	t.Parallel()
	raw := map[string]string{"Qualifier": "{{qualifier}}"}
	ctx := map[string]string{"qualifier": "bwapp"}
	got, err := cfnparams.Resolve(raw, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got["Qualifier"] != "bwapp" {
		t.Errorf("got %q, want %q", got["Qualifier"], "bwapp")
	}
}

func TestResolve_MultiplePlaceholders(t *testing.T) {
	t.Parallel()
	raw := map[string]string{"Combined": "{{qualifier}}-{{primary-region}}"}
	ctx := map[string]string{"qualifier": "bwapp", "primary-region": "eu-central-1"}
	got, err := cfnparams.Resolve(raw, ctx)
	if err != nil {
		t.Fatal(err)
	}
	want := "bwapp-eu-central-1"
	if got["Combined"] != want {
		t.Errorf("got %q, want %q", got["Combined"], want)
	}
}

func TestResolve_MixedStaticAndInterpolated(t *testing.T) {
	t.Parallel()
	raw := map[string]string{
		"Qualifier": "{{qualifier}}",
		"Repo":      "basewarphq/bw",
	}
	ctx := map[string]string{"qualifier": "bwapp"}
	got, err := cfnparams.Resolve(raw, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got["Qualifier"] != "bwapp" {
		t.Errorf("Qualifier: got %q, want %q", got["Qualifier"], "bwapp")
	}
	if got["Repo"] != "basewarphq/bw" {
		t.Errorf("Repo: got %q, want %q", got["Repo"], "basewarphq/bw")
	}
}

func TestResolve_UnknownKey(t *testing.T) {
	t.Parallel()
	raw := map[string]string{"Foo": "{{nonexistent}}"}
	ctx := map[string]string{"qualifier": "bwapp"}
	_, err := cfnparams.Resolve(raw, ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("error should mention unknown key, got: %v", err)
	}
}

func TestResolve_EmptyMap(t *testing.T) {
	t.Parallel()
	got, err := cfnparams.Resolve(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty map, got %v", got)
	}
}
