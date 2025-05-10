// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package helm

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
	"helm.sh/helm/v4/pkg/action"
	helmchart "helm.sh/helm/v4/pkg/chart/v2"
	helmchartutil "helm.sh/helm/v4/pkg/chart/v2/util"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/jaypipes/kube-inspect/debug"
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

	// The Helm Chart may specify a KubeVersion in its metadata that is
	// incompatible with the Kubernetes client version used in compiling the
	// Helm Go SDK. If this is the case, we need to pass an updated KubeVersion
	// installer option when rendering.
	if err := c.autoAdjustKubeVersion(ctx, installer); err != nil {
		return err
	}

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

// autoAdjustKubeVersion detects if the KubeVersion used by the Helm SDK is
// incompatible with the required KubeVersion specified in the Helm Chart
// metadata KubeVersion. If incompatible, the function figures out an
// appropriate KubeVersion and manually overrides the Install.KubeVersion
// option to that workable KubeVersion.
//
// See: https://github.com/jaypipes/kube-inspect/issues/2
// See: https://github.com/helm/helm/blob/3a94215585b91d5ac41ebb258e376aa11980b564/pkg/chartutil/capabilities.go#L31-L50
func (c *Chart) autoAdjustKubeVersion(
	ctx context.Context,
	installer *action.Install,
) error {
	hc := c.Chart
	if hc.Metadata.KubeVersion == "" {
		return nil
	}
	vc, err := semver.NewConstraint(hc.Metadata.KubeVersion)
	if err != nil {
		return err
	}
	debug.Printf(ctx, "chart kubeVersion constraint: %s\n", vc.String())
	dv, _ := semver.NewVersion(
		helmchartutil.DefaultCapabilities.KubeVersion.String(),
	)
	if !vc.Check(dv) {
		var uv *semver.Version
		constraintStr := strings.TrimSpace(vc.String())
		if strings.HasPrefix(constraintStr, "<=") || strings.HasPrefix(constraintStr, ">=") {
			wantVer := strings.TrimPrefix(
				strings.TrimPrefix(
					constraintStr, "<=",
				), ">=",
			)
			uv, err = semver.NewVersion(wantVer)
			if err != nil {
				return err
			}
		} else if strings.HasPrefix(constraintStr, "<") {
			wantVer := strings.TrimPrefix(constraintStr, "<")
			ov, err := semver.NewVersion(wantVer)
			if err != nil {
				return err
			}
			uv = semver.New(ov.Major(), ov.Minor()-1, ov.Patch(), "", "")
		} else if strings.HasPrefix(constraintStr, ">") {
			wantVer := strings.TrimPrefix(constraintStr, ">")
			ov, err := semver.NewVersion(wantVer)
			if err != nil {
				return err
			}
			uv = semver.New(ov.Major(), ov.Minor()+1, ov.Patch(), "", "")
		}
		newKV, _ := helmchartutil.ParseKubeVersion(uv.String())
		debug.Printf(
			ctx,
			"version check failed for default Helm SDK kubeVersion %q. "+
				"setting installer.KubeVersion manually to %q.\n",
			dv.String(), newKV.String(),
		)
		installer.KubeVersion = newKV
	}
	return nil
}
