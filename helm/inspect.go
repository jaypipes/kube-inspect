// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package helm

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"helm.sh/helm/v3/pkg/action"
	helmchart "helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/strvals"

	"github.com/jaypipes/kube-inspect/debug"
	"github.com/jaypipes/kube-inspect/kube"
)

// InspectOptions is a mechanism for you to control the inspection of a Helm
// Chart.
type InspectOptions struct {
	values map[string]any
}

type InspectOption func(opts *InspectOptions)

// WithValues allows passing values.yaml overrides to the Inspect function.
//
// The `vals` parameter should be a string or a map of string to interface.
//
// You may choose to pass a "strvals" single string, e.g. "pdb.create=true",
// instead of a nested map.
func WithValues(vals any) InspectOption {
	return func(opts *InspectOptions) {
		switch vals := vals.(type) {
		case string:
			opts.values, _ = strvals.Parse(vals)
		case map[string]any:
			opts.values = vals
		}
	}
}

// Inspect returns a `Chart` that describes a Helm Chart that has been rendered
// to actual Kubernetes resource manifests.
//
// The `subject` argument can be a filepath, a URL, a helm sdk-go `*Chart`
// struct, or an `io.Reader` pointing at either a directory or a compressed tar
// archive.
func Inspect(
	ctx context.Context,
	subject any,
	opt ...InspectOption,
) (*Chart, error) {
	opts := &InspectOptions{}
	for _, o := range opt {
		o(opts)
	}
	ctx = debug.PushTrace(ctx, "helm:inspect")
	defer debug.PopTrace(ctx)
	var err error
	var hc *helmchart.Chart
	switch subject := subject.(type) {
	case string:
		if strings.HasPrefix(subject, "http") {
			tf, err := fetchArchive(ctx, subject)
			if err != nil {
				return nil, err
			}
			defer os.Remove(tf.Name())
			hc, err = loader.LoadArchive(tf)
			if err != nil {
				return nil, fmt.Errorf("error loading archive: %w", err)
			}
		} else {
			hc, err = loader.Load(subject)
			if err != nil {
				return nil, err
			}

		}
	case *helmchart.Chart:
		if subject == nil {
			return nil, fmt.Errorf("passed nil helm sdk-go *Chart struct")
		}
		hc = subject
	case io.Reader:
		hc, err = loader.LoadArchive(subject)
		if err != nil {
			return nil, fmt.Errorf("error loading archive: %w", err)
		}
	default:
		return nil, fmt.Errorf(
			"unhandled type for inspect subject: %s (%T)",
			subject, subject,
		)
	}
	// Unfortunately, the helm sdk-go Release.Info.Resources map is empty when
	// "installing" in dry-run mode (which is necessary to render the templates
	// but not actually install anything). So we need to manually construct the
	// set of Kubernetes resources by processing the rendered multi-document
	// YAML manifest.
	manifest, err := manifestFromChart(ctx, hc, opts)
	if err != nil {
		return nil, err
	}
	resources, err := kube.ResourcesFromManifest(ctx, manifest)
	if err != nil {
		return nil, err
	}
	return &Chart{
		Chart:     hc,
		resources: resources,
	}, nil
}

// manifestFromChart accepts a helm sdk-go Chart object and returns a buffer
// containing a YAML document containing zero or more Kubernetes resource
// manifests that have been synthesized by running a dry-running install of the
// Helm Chart.
func manifestFromChart(
	ctx context.Context,
	hc *helmchart.Chart,
	opts *InspectOptions,
) (*bytes.Buffer, error) {
	ctx = debug.PushTrace(ctx, "helm:manifest-from-chart")
	defer debug.PopTrace(ctx)
	installer := action.NewInstall(&action.Configuration{})
	installer.ClientOnly = true
	installer.DryRun = true
	installer.ReleaseName = "kube-inspect"
	installer.IncludeCRDs = true
	installer.Namespace = "default"
	installer.DisableHooks = true
	if opts.values != nil {
		debug.Printf(ctx, "using value overrides: %v\n", opts.values)
	}
	release, err := installer.Run(hc, opts.values)
	if err != nil {
		return nil, err
	}
	return bytes.NewBuffer([]byte(release.Manifest)), nil
}

// fetchArchive reads the tarball at the supplied URL, copies it to a temporary
// file and returns the temporary file. callers are responsible for removing
// the temporary file.
func fetchArchive(
	ctx context.Context,
	url string,
) (*os.File, error) {
	ctx = debug.PushTrace(ctx, "helm:fetch-archive")
	defer debug.PopTrace(ctx)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-ok read from %q: %d", url, resp.StatusCode)
	}

	f, err := os.CreateTemp("", filepath.Base(url))
	if err != nil {
		return nil, err
	}
	io.Copy(f, resp.Body)
	f.Seek(0, 0)
	return f, nil
}
