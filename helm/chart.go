// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package helm

import (
	"bytes"
	"context"
	"fmt"

	"github.com/jaypipes/kube-inspect/debug"
	"helm.sh/helm/v3/pkg/action"
	helmchart "helm.sh/helm/v3/pkg/chart"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Chart describes an inspected Helm Chart that has been rendered to actual
// Kubernetes resources. This struct inherits from helm sdk-go's Chart struct
// and therefore exposes all of that struct's methods and metadata getters.
type Chart struct {
	*helmchart.Chart
	rendered    bool
	manifest    *bytes.Buffer
	inspectOpts *InspectOptions
	// resources is a slice of Kubernetes resources represented as
	// `unstructured.Unstructured` documents that was found in the
	// rendered/synthesized Helm Chart.
	resources []*unstructured.Unstructured
}

// render installs the Helm chart and sets the Chart.manifest to a buffer
// containing a YAML document containing zero or more Kubernetes resource
// manifests that have been synthesized by running a dry-running install of the
// Helm Chart.
func (c *Chart) render(
	ctx context.Context,
) error {
	hc := c.Chart
	if hc == nil {
		return fmt.Errorf("cannot render nil chart.")
	}
	if c.rendered {
		return nil
	}
	ctx = debug.PushTrace(ctx, "helm:chart:render")
	defer debug.PopTrace(ctx)
	installer := action.NewInstall(&action.Configuration{})
	installer.ClientOnly = true
	installer.DryRun = true
	installer.ReleaseName = "kube-inspect"
	installer.IncludeCRDs = true
	installer.Namespace = "default"
	installer.DisableHooks = true
	opts := c.inspectOpts
	if opts.values != nil {
		debug.Printf(ctx, "using value overrides: %v\n", opts.values)
	}
	release, err := installer.Run(hc, opts.values)
	if err != nil {
		return err
	}
	c.manifest = bytes.NewBuffer([]byte(release.Manifest))
	c.rendered = true
	return nil
}
