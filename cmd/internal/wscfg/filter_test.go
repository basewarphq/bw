package wscfg_test

import (
	"testing"

	"github.com/basewarphq/bw/cmd/internal/wscfg"
)

func checkResult(t *testing.T, got []wscfg.ProjectConfig, wantNames []string) {
	t.Helper()
	if len(got) != len(wantNames) {
		t.Fatalf("got %d projects, want %d", len(got), len(wantNames))
	}
	for i, name := range wantNames {
		if got[i].Name != name {
			t.Errorf("result[%d] = %q, want %q", i, got[i].Name, name)
		}
	}
	idx := make(map[string]int, len(got))
	for i, p := range got {
		idx[p.Name] = i
	}
	for _, p := range got {
		for _, dep := range p.DependsOn {
			depIdx, ok := idx[dep]
			if !ok {
				continue
			}
			if depIdx >= idx[p.Name] {
				t.Errorf("dependency %q (index %d) should appear before %q (index %d)", dep, depIdx, p.Name, idx[p.Name])
			}
		}
	}
}

func TestFilterProjectsIncludesDirectDeps(t *testing.T) {
	t.Parallel()
	projects := []wscfg.ProjectConfig{
		{Name: "lib", Dir: "lib"},
		{Name: "app", Dir: "app", DependsOn: []string{"lib"}},
	}
	got := wscfg.FilterProjects(projects, "app", false)
	checkResult(t, got, []string{"lib", "app"})
}

func TestFilterProjectsIncludesTransitiveDeps(t *testing.T) {
	t.Parallel()
	projects := []wscfg.ProjectConfig{
		{Name: "core", Dir: "core"},
		{Name: "lib", Dir: "lib", DependsOn: []string{"core"}},
		{Name: "app", Dir: "app", DependsOn: []string{"lib"}},
	}
	got := wscfg.FilterProjects(projects, "app", false)
	checkResult(t, got, []string{"core", "lib", "app"})
}

func TestFilterProjectsNoDeps(t *testing.T) {
	t.Parallel()
	projects := []wscfg.ProjectConfig{
		{Name: "lib", Dir: "lib"},
		{Name: "app", Dir: "app", DependsOn: []string{"lib"}},
	}
	got := wscfg.FilterProjects(projects, "app", true)
	checkResult(t, got, []string{"app"})
}

func TestFilterProjectsUnknownName(t *testing.T) {
	t.Parallel()
	projects := []wscfg.ProjectConfig{
		{Name: "lib", Dir: "lib"},
		{Name: "app", Dir: "app"},
	}
	got := wscfg.FilterProjects(projects, "nonexistent", false)
	checkResult(t, got, []string{"lib", "app"})
}

func TestFilterProjectsDiamondDeps(t *testing.T) {
	t.Parallel()
	projects := []wscfg.ProjectConfig{
		{Name: "core", Dir: "core"},
		{Name: "left", Dir: "left", DependsOn: []string{"core"}},
		{Name: "right", Dir: "right", DependsOn: []string{"core"}},
		{Name: "app", Dir: "app", DependsOn: []string{"left", "right"}},
	}
	got := wscfg.FilterProjects(projects, "app", false)
	if len(got) != 4 {
		t.Fatalf("got %d projects, want 4", len(got))
	}
	seen := make(map[string]int)
	for _, p := range got {
		seen[p.Name]++
	}
	if seen["core"] != 1 {
		t.Errorf("core appears %d times, want 1", seen["core"])
	}
	idx := make(map[string]int, len(got))
	for i, p := range got {
		idx[p.Name] = i
	}
	for _, p := range got {
		for _, dep := range p.DependsOn {
			if idx[dep] >= idx[p.Name] {
				t.Errorf("dependency %q (index %d) should appear before %q (index %d)", dep, idx[dep], p.Name, idx[p.Name])
			}
		}
	}
}

func TestFilterProjectsEmptyName(t *testing.T) {
	t.Parallel()
	projects := []wscfg.ProjectConfig{
		{Name: "lib", Dir: "lib"},
		{Name: "app", Dir: "app"},
	}
	got := wscfg.FilterProjects(projects, "", false)
	checkResult(t, got, []string{"lib", "app"})
}

func TestFilterProjectsNoDependencies(t *testing.T) {
	t.Parallel()
	projects := []wscfg.ProjectConfig{
		{Name: "lib", Dir: "lib"},
		{Name: "app", Dir: "app"},
	}
	got := wscfg.FilterProjects(projects, "app", false)
	checkResult(t, got, []string{"app"})
}
