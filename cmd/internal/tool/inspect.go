package tool

import (
	"context"
	"slices"
)

type Inspection struct {
	Name        string
	Description string
	Run         func(ctx context.Context, dir string, r NodeReporter) error
}

type InspectionProvider interface {
	Inspections() []Inspection
}

func RunInspections(ctx context.Context, p InspectionProvider, dir string, r NodeReporter) error {
	selected := InspectSelectionFrom(ctx)
	for _, insp := range p.Inspections() {
		if len(selected) > 0 && !slices.Contains(selected, insp.Name) {
			continue
		}
		r.Section(insp.Name)
		if err := insp.Run(ctx, dir, r); err != nil {
			return err
		}
	}
	return nil
}

func InspectionNames(t Tool) []string {
	p, ok := t.(InspectionProvider)
	if !ok {
		return nil
	}
	inspections := p.Inspections()
	names := make([]string, len(inspections))
	for i, insp := range inspections {
		names[i] = insp.Name
	}
	return names
}

type inspectSelectionKey struct{}

func WithInspectSelection(ctx context.Context, names []string) context.Context {
	return context.WithValue(ctx, inspectSelectionKey{}, names)
}

func InspectSelectionFrom(ctx context.Context) []string {
	v, _ := ctx.Value(inspectSelectionKey{}).([]string)
	return v
}
