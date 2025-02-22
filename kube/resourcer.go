// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package kube

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ResourceFilter represents a filtering expression for Resources returned by a
// call to `Resourcer.Resources()`
type ResourceFilter func(res *unstructured.Unstructured, _ int) bool

// Resourcer can return Resources that match zero or more filters.
type Resourcer interface {
	// Resources returns a slice of Kubernetes resources installed by the Helm
	// Chart that match a supplied filter.
	Resources(
		ctx context.Context,
		filters ...ResourceFilter,
	) ([]*unstructured.Unstructured, error)
}

// WithName returns a ResourceFilter that filters a Resource by
// `metadata.name`.
func WithName(name string) ResourceFilter {
	return func(res *unstructured.Unstructured, _ int) bool {
		return res.GetName() == name
	}
}

// WithKind returns a ResourceFilter that filters a Resource by
// `metadata.kind`.
func WithKind(kind string) ResourceFilter {
	return func(res *unstructured.Unstructured, _ int) bool {
		return res.GetKind() == kind
	}
}
