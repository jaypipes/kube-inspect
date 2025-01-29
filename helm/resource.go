// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package helm

import (
	"context"
	"fmt"
	"strings"

	"github.com/jaypipes/kube-inspect/debug"
	"github.com/jaypipes/kube-inspect/kube"
	"github.com/samber/lo"
	"github.com/santhosh-tekuri/jsonschema"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Resources returns a slice of Kubernetes resources installed by the Helm
// Chart that match a supplied filter.
func (c *Chart) Resources(
	ctx context.Context,
) ([]*unstructured.Unstructured, error) {
	if !c.rendered {
		if err := c.render(ctx); err != nil {
			return nil, err
		}
	}
	// Unfortunately, the helm sdk-go Release.Info.Resources map is empty when
	// "installing" in dry-run mode (which is necessary to render the templates
	// but not actually install anything). So we need to manually construct the
	// set of Kubernetes resources by processing the rendered multi-document
	// YAML manifest.
	resources, err := kube.ResourcesFromManifest(ctx, c.manifest)
	if err != nil {
		return nil, err
	}
	c.resources = resources
	return c.resources, nil
}

// OptionalResource contains information about a Kubernetes Resource that the
// Helm Chart may install.
type OptionalResource struct {
	// GroupKind is the Kubernetes GroupKind that was identified for the
	// Resource during inspection.
	GroupKind schema.GroupKind
	// DefaultName is the name of the Resource that would be rendered with the
	// default values collection.
	DefaultName string
	// ValueToggle is the strvals notation for the configuration value that
	// toggles the creation or enablement of this Resource. For example, assume
	// the very common practice of optionally creating a ServiceAccount
	// Resource when the `serviceAccount.create` value is set to "true", this
	// field would contain "serviceAccount.create=true"
	ValueToggle string
	// DefaultEnabled is true when the Resource is created/enabled when the
	// default values configuration is used during installation.
	DefaultEnabled bool
}

// OptionalResources returns a slice of OptionalResourceMeta objects that
// describe the optional Kubernetes Resources that the Helm Chart may install.
func (c *Chart) OptionalResources(
	ctx context.Context,
) ([]*OptionalResource, error) {
	hc := c.Chart
	if hc == nil {
		return nil, nil
	}
	ctx = debug.PushTrace(ctx, "helm:chart:optional-resources")
	defer debug.PopTrace(ctx)
	vs, err := c.loadValuesSchema(ctx)
	if err != nil {
		return nil, err
	}
	res := []*OptionalResource{}
	if vs != nil {
		// Look through the values schema property metadata for boolean
		// "resource enablement" configuration toggles
		for k, prop := range vs.Properties {
			res = append(res, collectOptionalResources(
				ctx, "", k, prop, nil)...,
			)
		}
	}
	return res, nil
}

var (
	resourceTogglePropNames = []string{"create", "enabled"}
)

// collectOptionalResources recursively searches through a `jsonschema.Schema`
// object's properties looking for "enabled" or "created" property names and
// returning the discovered OptionalResource structs.
func collectOptionalResources(
	ctx context.Context,
	dottedKey string,
	propName string,
	prop *jsonschema.Schema,
	parent *jsonschema.Schema,
) []*OptionalResource {
	fullKey := propName
	if dottedKey != "" {
		fullKey = dottedKey + "." + propName
	}
	traceName := fmt.Sprintf(
		"helm:chart:collect-optional-resource (%s)", fullKey,
	)
	ctx = debug.PushTrace(ctx, traceName)
	defer debug.PopTrace(ctx)
	if likelyResourceToggle(propName, prop) {
		return []*OptionalResource{
			{
				ValueToggle:    fullKey,
				DefaultEnabled: true,
			},
		}
	}
	res := []*OptionalResource{}
	for k, p := range prop.Properties {
		res = append(res, collectOptionalResources(
			ctx, fullKey, k, p, prop)...,
		)
	}
	return res
}

func likelyResourceToggle(
	propName string,
	prop *jsonschema.Schema,
) bool {
	if len(prop.Types) > 1 || prop.Types[0] != "boolean" {
		return false
	}
	return lo.Contains(resourceTogglePropNames, strings.ToLower(propName))
}
