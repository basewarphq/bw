package dag_test

import (
	"context"
	"sync"
	"testing"

	"github.com/basewarphq/bw/cmd/internal/dag"
	"github.com/basewarphq/bw/cmd/internal/tool"
	"github.com/basewarphq/bw/cmd/internal/tool/gotool"
	"github.com/basewarphq/bw/cmd/internal/tool/shelltool"
	"github.com/basewarphq/bw/cmd/internal/wscfg"
	tfdag "github.com/sourcegraph/tf-dag/dag"
)

func newTestRegistry() *tool.Registry {
	reg := tool.NewRegistry()
	reg.Register(shelltool.New())
	reg.Register(gotool.New())
	return reg
}

func collectNodes(graph *tfdag.AcyclicGraph) []*dag.Node {
	var nodes []*dag.Node
	for _, vertex := range graph.Vertices() {
		node, ok := vertex.(*dag.Node)
		if ok {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

func nodeNames(graph *tfdag.AcyclicGraph) map[string]struct{} {
	names := make(map[string]struct{})
	for _, node := range collectNodes(graph) {
		names[node.Name()] = struct{}{}
	}
	return names
}

func hasEdge(graph *tfdag.AcyclicGraph, from, to string) bool {
	var fromNode, toNode *dag.Node
	for _, node := range collectNodes(graph) {
		if node.Name() == from {
			fromNode = node
		}
		if node.Name() == to {
			toNode = node
		}
	}
	if fromNode == nil || toNode == nil {
		return false
	}
	for _, edge := range graph.Edges() {
		if edge.Source() == fromNode && edge.Target() == toNode {
			return true
		}
	}
	return false
}

func TestBuildSingleProjectFmt(t *testing.T) {
	t.Parallel()
	reg := newTestRegistry()
	projects := []wscfg.ProjectConfig{
		{Name: "app", Dir: "app", Tools: []string{"go", "shell"}},
	}

	graph, err := dag.Build(projects, reg, "/ws", []tool.Step{tool.StepFmt})
	if err != nil {
		t.Fatal(err)
	}

	names := nodeNames(graph)
	if _, ok := names["app:fmt:go"]; !ok {
		t.Error("expected node app:fmt:go")
	}
	if _, ok := names["app:fmt:shell"]; !ok {
		t.Error("expected node app:fmt:shell")
	}
	if len(names) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(names))
	}
}

func TestBuildStepOrdering(t *testing.T) {
	t.Parallel()
	reg := newTestRegistry()
	projects := []wscfg.ProjectConfig{
		{Name: "app", Dir: "app", Tools: []string{"go"}},
	}

	graph, err := dag.Build(projects, reg, "/ws", tool.AllDevCheckSteps)
	if err != nil {
		t.Fatal(err)
	}

	if !hasEdge(graph, "app:gen:go", "app:fmt:go") {
		t.Error("expected edge gen -> fmt")
	}
	if !hasEdge(graph, "app:fmt:go", "app:lint:go") {
		t.Error("expected edge fmt -> lint")
	}
	if !hasEdge(graph, "app:lint:go", "app:compiles:go") {
		t.Error("expected edge lint -> compiles")
	}
	if !hasEdge(graph, "app:compiles:go", "app:unit-test:go") {
		t.Error("expected edge compiles -> unit-test")
	}
}

func TestBuildSkipsUnsupportedSteps(t *testing.T) {
	t.Parallel()
	reg := newTestRegistry()
	projects := []wscfg.ProjectConfig{
		{Name: "app", Dir: "app", Tools: []string{"shell"}},
	}

	graph, err := dag.Build(projects, reg, "/ws", tool.AllDevCheckSteps)
	if err != nil {
		t.Fatal(err)
	}

	names := nodeNames(graph)
	if _, ok := names["app:fmt:shell"]; !ok {
		t.Error("expected node app:fmt:shell")
	}
	if _, ok := names["app:lint:shell"]; !ok {
		t.Error("expected node app:lint:shell")
	}
	if _, ok := names["app:gen:shell"]; ok {
		t.Error("shell should not have gen step")
	}
	if _, ok := names["app:compiles:shell"]; ok {
		t.Error("shell should not have compiles step")
	}
	if _, ok := names["app:unit-test:shell"]; ok {
		t.Error("shell should not have unit-test step")
	}
	if !hasEdge(graph, "app:fmt:shell", "app:lint:shell") {
		t.Error("expected edge fmt -> lint (skipping unsupported gen)")
	}
}

func TestBuildProjectDependencies(t *testing.T) {
	t.Parallel()
	reg := newTestRegistry()
	projects := []wscfg.ProjectConfig{
		{Name: "lib", Dir: "lib", Tools: []string{"go"}},
		{Name: "app", Dir: "app", Tools: []string{"go"}, DependsOn: []string{"lib"}},
	}

	graph, err := dag.Build(projects, reg, "/ws", []tool.Step{tool.StepLint})
	if err != nil {
		t.Fatal(err)
	}

	if !hasEdge(graph, "lib:lint:go", "app:lint:go") {
		t.Error("expected project dependency edge lib:lint:go -> app:lint:go")
	}
}

func TestBuildUnknownToolErrors(t *testing.T) {
	t.Parallel()
	reg := newTestRegistry()
	projects := []wscfg.ProjectConfig{
		{Name: "app", Dir: "app", Tools: []string{"nonexistent"}},
	}

	_, err := dag.Build(projects, reg, "/ws", []tool.Step{tool.StepFmt})
	if err == nil {
		t.Error("expected error for unknown tool")
	}
}

func TestBuildUnknownProjectDepErrors(t *testing.T) {
	t.Parallel()
	reg := newTestRegistry()
	projects := []wscfg.ProjectConfig{
		{Name: "app", Dir: "app", Tools: []string{"go"}, DependsOn: []string{"missing"}},
	}

	_, err := dag.Build(projects, reg, "/ws", []tool.Step{tool.StepFmt})
	if err == nil {
		t.Error("expected error for unknown project dependency")
	}
}

func TestExecuteRunsAllNodes(t *testing.T) {
	t.Parallel()
	reg := tool.NewRegistry()
	mock := &mockTool{name: "mock"}
	reg.Register(mock)

	projects := []wscfg.ProjectConfig{
		{Name: "app", Dir: "/tmp", Tools: []string{"mock"}},
	}

	graph, err := dag.Build(projects, reg, "/", []tool.Step{tool.StepFmt, tool.StepLint})
	if err != nil {
		t.Fatal(err)
	}

	err = dag.Execute(context.Background(), graph)
	if err != nil {
		t.Fatal(err)
	}

	mock.mu.Lock()
	defer mock.mu.Unlock()
	if mock.fmtCalls != 1 {
		t.Errorf("expected 1 fmt call, got %d", mock.fmtCalls)
	}
	if mock.lintCalls != 1 {
		t.Errorf("expected 1 lint call, got %d", mock.lintCalls)
	}
}

type mockTool struct {
	name      string
	mu        sync.Mutex
	fmtCalls  int
	lintCalls int
}

func (m *mockTool) Name() string           { return m.name }
func (m *mockTool) Dependencies() []string { return nil }

func (m *mockTool) Fmt(_ context.Context, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.fmtCalls++
	return nil
}

func (m *mockTool) Lint(_ context.Context, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lintCalls++
	return nil
}
