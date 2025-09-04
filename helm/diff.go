// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package helm

import (
	"context"
	"fmt"

	"github.com/jaypipes/kube-inspect/diff"
	"github.com/jaypipes/kube-inspect/kube"
	"gopkg.in/yaml.v3"
)

type ChartDiff struct {
	// Resources describes the synthesized Kubernetes resources/manifests that
	// are different between the Charts.
	Resources diff.ResourcesDiff `yaml:"resources"`
	// Values describes the values.yaml fields that are different between the
	// Charts.
	Values diff.Diff `yaml:"values"`
}

// Diff returns a struct that represents the difference between this Chart and
// another Chart.
func (c *Chart) Diff(
	ctx context.Context,
	other *Chart,
	opt ...diff.DiffOption,
) (*ChartDiff, error) {
	opts := &diff.DiffOptions{}
	for _, o := range opt {
		o(opts)
	}

	resDiff, err := resourcesDiff(ctx, c, other)
	if err != nil {
		return nil, fmt.Errorf("failed to create resources diff: %w", err)
	}
	valsDiff, err := valuesDiff(c, other)
	if err != nil {
		return nil, fmt.Errorf("failed to create values diff: %w", err)
	}

	return &ChartDiff{
		Resources: *resDiff,
		Values:    *valsDiff,
	}, nil
}

func valuesDiff(
	a *Chart,
	b *Chart,
) (*diff.Diff, error) {
	aVals, err := yaml.Marshal(a.Values)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal chart A values: %w", err)
	}
	bVals, err := yaml.Marshal(b.Values)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal chart B values: %w", err)
	}
	return diff.New(aVals, bVals)
}

func resourcesDiff(
	ctx context.Context,
	a *Chart,
	b *Chart,
) (*diff.ResourcesDiff, error) {
	ars, err := a.Resources(ctx)
	if err != nil {
		return nil, err
	}
	brs, err := b.Resources(ctx)
	if err != nil {
		return nil, err
	}
	return kube.DiffResources(ars, brs)
}
