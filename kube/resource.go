// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package kube

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"

	"github.com/jaypipes/kube-inspect/debug"
	"github.com/jaypipes/kube-inspect/diff"
	"github.com/samber/lo"
)

// ResourcesFromManifest processes the supplied buffer containing a YAML
// document with zero or more Kubernetes resource manifests and returns a slice
// of unstructured.Unstructured structs representing those Kubernetes
// resources.
func ResourcesFromManifest(
	ctx context.Context,
	manifest *bytes.Buffer,
) ([]*unstructured.Unstructured, error) {
	ctx = debug.PushTrace(ctx, "kube:resources-from-manifest")
	defer debug.PopTrace(ctx)
	res := []*unstructured.Unstructured{}
	reader := bytes.NewReader(manifest.Bytes())
	decoder := yamlutil.NewYAMLOrJSONDecoder(reader, 100)
	parser := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	for {
		var rawObj runtime.RawExtension
		if err := decoder.Decode(&rawObj); err != nil {
			break
		}
		if len(rawObj.Raw) == 0 {
			continue
		}

		obj, gvk, err := parser.Decode(rawObj.Raw, nil, nil)
		if err != nil {
			return nil, err
		}
		usMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			return nil, err
		}

		us := &unstructured.Unstructured{Object: usMap}
		kind := gvk.GroupKind()
		name, _, err := unstructured.NestedString(us.Object, "metadata", "name")
		if err != nil {
			return nil, err
		}
		debug.Printf(ctx, "identified resource %s (%s)\n", name, kind)
		res = append(res, us)
	}

	return res, nil
}

type kindNameResourceMap map[string]map[string]*unstructured.Unstructured

// DiffResources returns the `diff.ResourcesDiff` that describes the
// differences between two supplied slices of resources.
//
// The two arguments should be []runtime.Object,
// *unstructured.UnstructuredList, or []*unstructured.Unstructured
func DiffResources(
	a, b any,
) (*diff.ResourcesDiff, error) {
	aResources, err := ToSliceUnstructured(a)
	if err != nil {
		return nil, err
	}
	bResources, err := ToSliceUnstructured(b)
	if err != nil {
		return nil, err
	}
	aResGroups := resourcesByGroupVersionKindAndName(aResources)
	bResGroups := resourcesByGroupVersionKindAndName(bResources)

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
				changes[fmt.Sprintf("%s/%s", aKind, aName)] = *report
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

	return &diff.ResourcesDiff{
		Changed:   changes,
		Added:     additions,
		Removed:   removals,
		Unchanged: unchanged,
	}, nil
}

// resourcesByGroupVersionKindAndName returns a map, keyed by Resource Kind prefixed
// with group version, of maps, keyed by Resource Name, of Kubernetes Resources. The
// resource name will be stripped of the "kube-inspect-" prefix that we tack on while
// templating the Helm release.
func resourcesByGroupVersionKindAndName(
	rs []*unstructured.Unstructured,
) kindNameResourceMap {
	res := kindNameResourceMap{}
	for _, r := range rs {
		groupVersionKind := groupVersionKindString(r.GroupVersionKind())
		if _, ok := res[groupVersionKind]; !ok {
			res[groupVersionKind] = map[string]*unstructured.Unstructured{}
		}
		if after, ok := strings.CutPrefix(r.GetName(), "kube-inspect-"); ok {
			r.SetName(after)
		}
		res[groupVersionKind][r.GetName()] = r
	}
	return res
}

// groupVersionKindString converts a GroupVersionKind into a string representation
// where each component is seperated by a /
func groupVersionKindString(gvk schema.GroupVersionKind) string {
	if gvk.Group == "" {
		return fmt.Sprintf("%s/%s", gvk.Version, gvk.Kind)
	} else {
		return fmt.Sprintf("%s/%s/%s", gvk.Group, gvk.Version, gvk.Kind)
	}
}
