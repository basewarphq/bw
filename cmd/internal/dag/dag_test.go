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

func dependsOn(graph *tfdag.AcyclicGraph, waiter, dep string) bool {
	var waiterNode, depNode *dag.Node
	for _, node := range collectNodes(graph) {
		if node.Name() == waiter {
			waiterNode = node
		}
		if node.Name() == dep {
			depNode = node
		}
	}
	if waiterNode == nil || depNode == nil {
		return false
	}
	for _, edge := range graph.Edges() {
		if edge.Source() == waiterNode && edge.Target() == depNode {
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

	graph, err := dag.Build(projects, reg, &wscfg.Config{Root: "/ws"}, []tool.Step{tool.StepFmt})
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

	graph, err := dag.Build(projects, reg, &wscfg.Config{Root: "/ws"}, tool.PreflightSteps)
	if err != nil {
		t.Fatal(err)
	}

	if !dependsOn(graph, "app:fmt:go", "app:gen:go") {
		t.Error("expected fmt to depend on gen")
	}
	if !dependsOn(graph, "app:lint:go", "app:fmt:go") {
		t.Error("expected lint to depend on fmt")
	}
	if !dependsOn(graph, "app:build:go", "app:lint:go") {
		t.Error("expected build to depend on lint")
	}
	if !dependsOn(graph, "app:unit-test:go", "app:build:go") {
		t.Error("expected unit-test to depend on build")
	}
}

func TestBuildSkipsUnsupportedSteps(t *testing.T) {
	t.Parallel()
	reg := newTestRegistry()
	projects := []wscfg.ProjectConfig{
		{Name: "app", Dir: "app", Tools: []string{"shell"}},
	}

	graph, err := dag.Build(projects, reg, &wscfg.Config{Root: "/ws"}, tool.PreflightSteps)
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
	if _, ok := names["app:build:shell"]; ok {
		t.Error("shell should not have build step")
	}
	if _, ok := names["app:unit-test:shell"]; ok {
		t.Error("shell should not have unit-test step")
	}
	if !dependsOn(graph, "app:lint:shell", "app:fmt:shell") {
		t.Error("expected lint to depend on fmt (skipping unsupported gen)")
	}
}

func TestBuildProjectDependencies(t *testing.T) {
	t.Parallel()
	reg := newTestRegistry()
	projects := []wscfg.ProjectConfig{
		{Name: "lib", Dir: "lib", Tools: []string{"go"}},
		{Name: "app", Dir: "app", Tools: []string{"go"}, DependsOn: []string{"lib"}},
	}

	graph, err := dag.Build(projects, reg, &wscfg.Config{Root: "/ws"}, []tool.Step{tool.StepLint})
	if err != nil {
		t.Fatal(err)
	}

	if !dependsOn(graph, "app:lint:go", "lib:lint:go") {
		t.Error("expected app:lint:go to depend on lib:lint:go")
	}
}

func TestBuildUnknownToolErrors(t *testing.T) {
	t.Parallel()
	reg := newTestRegistry()
	projects := []wscfg.ProjectConfig{
		{Name: "app", Dir: "app", Tools: []string{"nonexistent"}},
	}

	_, err := dag.Build(projects, reg, &wscfg.Config{Root: "/ws"}, []tool.Step{tool.StepFmt})
	if err == nil {
		t.Error("expected error for unknown tool")
	}
}

func TestBuildUnknownProjectDepSkipped(t *testing.T) {
	t.Parallel()
	reg := newTestRegistry()
	projects := []wscfg.ProjectConfig{
		{Name: "app", Dir: "app", Tools: []string{"go"}, DependsOn: []string{"missing"}},
	}

	graph, err := dag.Build(projects, reg, &wscfg.Config{Root: "/ws"}, []tool.Step{tool.StepFmt})
	if err != nil {
		t.Fatal(err)
	}

	names := nodeNames(graph)
	if _, ok := names["app:fmt:go"]; !ok {
		t.Error("expected node app:fmt:go")
	}
}

func TestBuildFilteredProjectWithDeps(t *testing.T) {
	t.Parallel()
	reg := newTestRegistry()
	projects := []wscfg.ProjectConfig{
		{Name: "lib", Dir: "lib", Tools: []string{"go"}},
		{Name: "app", Dir: "app", Tools: []string{"go"}, DependsOn: []string{"lib"}},
	}

	graph, err := dag.Build(projects, reg, &wscfg.Config{Root: "/ws"}, []tool.Step{tool.StepLint})
	if err != nil {
		t.Fatal(err)
	}

	names := nodeNames(graph)
	if _, ok := names["app:lint:go"]; !ok {
		t.Error("expected node app:lint:go")
	}
	if _, ok := names["lib:lint:go"]; !ok {
		t.Error("expected node lib:lint:go")
	}
	if !dependsOn(graph, "app:lint:go", "lib:lint:go") {
		t.Error("expected app:lint:go to depend on lib:lint:go")
	}
}

func TestBuildFilteredProjectNoDepsFlag(t *testing.T) {
	t.Parallel()
	reg := newTestRegistry()
	projects := []wscfg.ProjectConfig{
		{Name: "app", Dir: "app", Tools: []string{"go"}, DependsOn: []string{"lib"}},
	}

	graph, err := dag.Build(projects, reg, &wscfg.Config{Root: "/ws"}, []tool.Step{tool.StepLint})
	if err != nil {
		t.Fatal(err)
	}

	names := nodeNames(graph)
	if _, ok := names["app:lint:go"]; !ok {
		t.Error("expected node app:lint:go")
	}
	if len(names) != 1 {
		t.Errorf("expected 1 node, got %d", len(names))
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

	graph, err := dag.Build(projects, reg, &wscfg.Config{Root: "/"}, []tool.Step{tool.StepFmt, tool.StepLint})
	if err != nil {
		t.Fatal(err)
	}

	err = dag.Execute(context.Background(), graph, noopReporter{})
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

type noopReporter struct{}

func (noopReporter) ForNode(_, _, _ string) tool.NodeReporter { return noopNodeReporter{} }

type noopNodeReporter struct{}

func (noopNodeReporter) Section(_ string)               {}
func (noopNodeReporter) Table(_ []string, _ [][]string) {}
func (noopNodeReporter) Error(_ string)                 {}

type mockTool struct {
	name      string
	mu        sync.Mutex
	fmtCalls  int
	lintCalls int
}

func (m *mockTool) Name() string        { return m.name }
func (m *mockTool) RunsAfter() []string { return nil }

func (m *mockTool) Fmt(_ context.Context, _ string, _ tool.NodeReporter) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.fmtCalls++
	return nil
}

func (m *mockTool) Lint(_ context.Context, _ string, _ tool.NodeReporter) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lintCalls++
	return nil
}

type configMockTool struct {
	name           string
	mu             sync.Mutex
	receivedConfig any
}

func (m *configMockTool) Name() string        { return m.name }
func (m *configMockTool) RunsAfter() []string { return nil }

func (m *configMockTool) Fmt(ctx context.Context, _ string, _ tool.NodeReporter) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.receivedConfig = tool.ToolConfigFrom[string](ctx)
	return nil
}

func TestExecutePassesToolConfig(t *testing.T) {
	t.Parallel()
	reg := tool.NewRegistry()
	mock := &configMockTool{name: "cfgmock"}
	reg.Register(mock)

	cfg := &wscfg.Config{
		Root: "/",
		DecodedToolConfigs: map[string]map[string]any{
			"app": {"cfgmock": "test-profile"},
		},
	}

	projects := []wscfg.ProjectConfig{
		{Name: "app", Dir: "/tmp", Tools: []string{"cfgmock"}},
	}

	graph, err := dag.Build(projects, reg, cfg, []tool.Step{tool.StepFmt})
	if err != nil {
		t.Fatal(err)
	}

	err = dag.Execute(context.Background(), graph, noopReporter{})
	if err != nil {
		t.Fatal(err)
	}

	mock.mu.Lock()
	defer mock.mu.Unlock()
	got, ok := mock.receivedConfig.(*string)
	if !ok || got == nil {
		t.Fatalf("expected *string config, got %T", mock.receivedConfig)
	}
	if *got != "test-profile" {
		t.Errorf("expected config %q, got %q", "test-profile", *got)
	}
}
