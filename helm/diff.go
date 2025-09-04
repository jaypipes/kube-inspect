// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package helm

import (
	"context"
	"fmt"
	"strings"

	"github.com/jaypipes/kube-inspect/diff"
	"github.com/samber/lo"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type ResourcesDiff struct {
	// Changed is a map, keyed by full resource name
	// (APIGroupVersion/ResourceName) of resources different between Charts A
	// and B.
	Changed map[string]diff.Diff
	// Added contains resources present in Chart B that are not present in
	// Chart A.
	Added []*unstructured.Unstructured
	// Removed contains resources present in Chart A that are not present in
	// Chart B.
	Removed []*unstructured.Unstructured
	// Unchanged contains resources that are exactly the same in both Charts.
	Unchanged []*unstructured.Unstructured
}

type ChartDiff struct {
	// Resources describes the synthesized Kubernetes resources/manifests that
	// are different between the Charts.
	Resources ResourcesDiff `yaml:"resources"`
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
) (*ResourcesDiff, error) {
	aResGroups, err := resourceGroupsFromChart(ctx, a)
	if err != nil {
		return nil, fmt.Errorf("failed to get resources from chart A: %w", err)
	}
	bResGroups, err := resourceGroupsFromChart(ctx, b)
	if err != nil {
		return nil, fmt.Errorf("failed to get resources from chart B: %w", err)
	}
	var additions, removals, unchanged []*unstructured.Unstructured
	changes := map[string]diff.Diff{}

	for aKind, aNameResources := range aResGroups {
		if _, ok := bResGroups[aKind]; !ok {
			removals = append(removals, lo.Values(aNameResources)...)
			continue
		}

		for aName, aRes := range aNameResources {
			bRes, ok := bResGroups[aKind][aName]
			if !ok {
				removals = append(removals, aRes)
				continue
			}
			report, err := diff.New(aRes, bRes)
			if err != nil {
				return nil, fmt.Errorf("failed to get dyff report: %w", err)
			}
			if len(report.Diffs) > 0 {
				changes[aName] = *report
			} else {
				unchanged = append(unchanged, aRes)
			}
		}
	}

	for bKind, bNameResources := range bResGroups {
		if _, ok := aResGroups[bKind]; !ok {
			removals = append(removals, lo.Values(bNameResources)...)
			continue
		}
		for bName, bRes := range bNameResources {
			if _, ok := aResGroups[bKind][bName]; !ok {
				additions = append(additions, bRes)
			}
		}
	}

	return &ResourcesDiff{
		Changed:   changes,
		Added:     additions,
		Removed:   removals,
		Unchanged: unchanged,
	}, nil
}

// resourceGroupsFromChart returns a map, keyed by Resource Kind, of maps,
// keyed by Resource Name, of Kubernetes Resources. The resource name will be
// stripped of the "kube-inspect-" prefix that we tack on while templating the
// Helm release.
func resourceGroupsFromChart(
	ctx context.Context,
	chart *Chart,
) (map[string]map[string]*unstructured.Unstructured, error) {
	rs, err := chart.Resources(ctx)
	if err != nil {
		return nil, err
	}

	res := map[string]map[string]*unstructured.Unstructured{}
	for _, r := range rs {
		if _, ok := res[r.GetKind()]; !ok {
			res[r.GetKind()] = map[string]*unstructured.Unstructured{}
		}
		if strings.HasPrefix(r.GetName(), "kube-inspect-") {
			r.SetName(strings.TrimPrefix(r.GetName(), "kube-inspect-"))
		}
		res[r.GetKind()][r.GetName()] = r
	}
	return res, nil
}
