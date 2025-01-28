// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package helm

import (
	helmchart "helm.sh/helm/v3/pkg/chart"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Chart describes an inspected Helm Chart that has been rendered to actual
// Kubernetes resources. This struct inherits from helm sdk-go's Chart struct
// and therefore exposes all of that struct's methods and metadata getters.
type Chart struct {
	*helmchart.Chart
	// resources is a slice of Kubernetes resources represented as
	// `unstructured.Unstructured` documents that was found in the
	// rendered/synthesized Helm Chart.
	resources []*unstructured.Unstructured
}

// Resources returns a slice of Kubernetes resources installed by the Helm
// Chart that match a supplied filter.
func (c *Chart) Resources() []*unstructured.Unstructured {
	res := []*unstructured.Unstructured{}
	for _, r := range c.resources {
		res = append(res, r)
	}
	return res
}
