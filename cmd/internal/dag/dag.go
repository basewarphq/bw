package dag

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/basewarphq/bw/cmd/internal/tool"
	"github.com/basewarphq/bw/cmd/internal/wscfg"
	"github.com/cockroachdb/errors"
	tfdag "github.com/sourcegraph/tf-dag/dag"
)

type Node struct {
	Project string
	Step    tool.Step
	Tool    tool.Tool
	Dir     string
	Config  any
}

func (n *Node) Name() string {
	return fmt.Sprintf("%s:%s:%s", n.Project, n.Step, n.Tool.Name())
}

type nodeKey struct {
	project string
	step    tool.Step
	tool    string
}

type builder struct {
	graph    tfdag.AcyclicGraph
	nodes    map[nodeKey]*Node
	registry *tool.Registry
	cfg      *wscfg.Config
	steps    []tool.Step
}

func Build(
	projects []wscfg.ProjectConfig,
	registry *tool.Registry,
	cfg *wscfg.Config,
	steps []tool.Step,
) (*tfdag.AcyclicGraph, error) {
	projects = wscfg.FilterProjects(projects, cfg.ProjectFilter, cfg.NoDeps)

	bld := &builder{
		nodes:    make(map[nodeKey]*Node),
		registry: registry,
		cfg:      cfg,
		steps:    steps,
	}

	if err := bld.createNodes(projects); err != nil {
		return nil, err
	}
	bld.addStepEdges(projects)
	bld.addToolDepEdges(projects)
	bld.addProjectDepEdges(projects)

	bld.graph.TransitiveReduction()

	if cycles := bld.graph.Cycles(); len(cycles) > 0 {
		return nil, errors.Newf("dependency cycle detected in execution graph")
	}

	return &bld.graph, nil
}

func (bld *builder) resolveTools(proj wscfg.ProjectConfig) ([]tool.Tool, error) {
	tools := make([]tool.Tool, 0, len(proj.Tools))
	for _, toolName := range proj.Tools {
		resolved, err := bld.registry.Get(toolName)
		if err != nil {
			return nil, errors.Wrapf(err, "project %q", proj.Name)
		}
		tools = append(tools, resolved)
	}
	return tools, nil
}

func (bld *builder) createNodes(projects []wscfg.ProjectConfig) error {
	for _, proj := range projects {
		projDir := filepath.Join(bld.cfg.Root, proj.Dir)
		projTools, err := bld.resolveTools(proj)
		if err != nil {
			return err
		}
		for _, step := range bld.steps {
			for _, tl := range projTools {
				if !tool.SupportsStep(tl, step) {
					continue
				}
				node := &Node{
					Project: proj.Name,
					Step:    step,
					Tool:    tl,
					Dir:     projDir,
					Config:  bld.cfg.ProjectToolConfig(proj.Name, tl.Name()),
				}
				key := nodeKey{proj.Name, step, tl.Name()}
				bld.nodes[key] = node
				bld.graph.Add(node)
			}
		}
	}
	return nil
}

func (bld *builder) addStepEdges(projects []wscfg.ProjectConfig) {
	for _, proj := range projects {
		for _, tl := range proj.Tools {
			for idx := 1; idx < len(bld.steps); idx++ {
				curr := bld.nodes[nodeKey{proj.Name, bld.steps[idx], tl}]
				if curr == nil {
					continue
				}
				for back := idx - 1; back >= 0; back-- {
					prev := bld.nodes[nodeKey{proj.Name, bld.steps[back], tl}]
					if prev != nil {
						bld.graph.Connect(tfdag.BasicEdge(curr, prev))
						break
					}
				}
			}
		}
	}
}

func (bld *builder) addToolDepEdges(projects []wscfg.ProjectConfig) {
	for _, proj := range projects {
		projToolSet := make(map[string]struct{}, len(proj.Tools))
		for _, tl := range proj.Tools {
			projToolSet[tl] = struct{}{}
		}

		for _, toolName := range proj.Tools {
			resolved, err := bld.registry.Get(toolName)
			if err != nil {
				continue
			}
			for _, depName := range resolved.RunsAfter() {
				if _, ok := projToolSet[depName]; !ok {
					continue
				}
				for _, step := range bld.steps {
					src := bld.nodes[nodeKey{proj.Name, step, depName}]
					dst := bld.nodes[nodeKey{proj.Name, step, toolName}]
					if src != nil && dst != nil {
						bld.graph.Connect(tfdag.BasicEdge(dst, src))
					}
				}
			}
		}
	}
}

func (bld *builder) addProjectDepEdges(projects []wscfg.ProjectConfig) {
	projByName := make(map[string]wscfg.ProjectConfig, len(projects))
	for _, proj := range projects {
		projByName[proj.Name] = proj
	}

	for _, proj := range projects {
		for _, depName := range proj.DependsOn {
			depProj, ok := projByName[depName]
			if !ok {
				continue
			}
			for _, step := range bld.steps {
				for _, toolName := range proj.Tools {
					dst := bld.nodes[nodeKey{proj.Name, step, toolName}]
					if dst == nil {
						continue
					}
					for _, depTool := range depProj.Tools {
						src := bld.nodes[nodeKey{depName, step, depTool}]
						if src != nil {
							bld.graph.Connect(tfdag.BasicEdge(dst, src))
						}
					}
				}
			}
		}
	}
}

func Execute(ctx context.Context, graph *tfdag.AcyclicGraph, reporter tool.Reporter) error {
	return graph.Walk(func(vertex tfdag.Vertex) error {
		node, ok := vertex.(*Node)
		if !ok {
			return errors.Newf("unexpected vertex type: %T", vertex)
		}
		nodeCtx := ctx
		if node.Config != nil {
			nodeCtx = tool.WithToolConfig(nodeCtx, node.Config)
		}
		r := reporter.ForNode(node.Project, node.Step.String(), node.Tool.Name())
		if err := tool.RunStep(nodeCtx, node.Tool, node.Step, node.Dir, r); err != nil {
			return errors.Wrapf(err, "%s", node.Name())
		}
		return nil
	})
}
