// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package helm

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"helm.sh/helm/v3/pkg/action"
	helmchart "helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/strvals"

	"github.com/jaypipes/kube-inspect/debug"
)

// InspectOptions is a mechanism for you to control the inspection of a Helm
// Chart.
type InspectOptions struct {
	// chartName specifies the name of the chart. Only used when the user
	// specified an OCI registry URL as the subject parameter for Inspect().
	chartName string
	// chartVersion specifies the version of the chart. Only used when the user
	// specified an OCI registry URL as the subject parameter for Inspect().
	chartVersion string
	// registryClient specifies an optional Helm registry client object to use
	// when fetching/pulling the chart. Only used when the user specified an
	// OCI registry URL as the subject parameter for Inspect().
	registryClient *registry.Client

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

// WithChartName adds a chart name specifier to the Inspect chart fetching
// operation. Only used when pulling from an OCI registry (when subject is an
// OCI registry URL).
func WithChartName(name string) InspectOption {
	return func(opts *InspectOptions) {
		opts.chartName = name
	}
}

// WithChartVersion adds a chart version specifier to the Inspect chart
// fetching operation. Only used when pulling from an OCI registry (when
// subject is an OCI registry URL).
func WithChartVersion(version string) InspectOption {
	return func(opts *InspectOptions) {
		opts.chartVersion = version
	}
}

// WithRegistryClient adds a Helm Registry client to the Inspect chart fetching
// operation. Only used when pulling from an OCI registry (when subject is an
// OCI registry URL).
func WithRegistryClient(c *registry.Client) InspectOption {
	return func(opts *InspectOptions) {
		opts.registryClient = c
	}
}

// Inspect returns a `Chart` that describes a Helm Chart that has been rendered
// to actual Kubernetes resource manifests.
//
// The `subject` argument can be a filepath, a URL, a helm sdk-go `*Chart`
// struct, or an `io.Reader` pointing at either a directory or a compressed tar
// archive. If `subject` is a an OCI registry URL, then the function will
// attempt to pull the Helm Chart from the supplied OCI registry and unpack it
// to a local directory.
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
		if registry.IsOCI(subject) {
			rc := opts.registryClient
			if rc == nil {
				rc, err = registry.NewClient()
				if err != nil {
					return nil, fmt.Errorf("failed to create default registry client: %w", err)
				}
			}
			chartVersion := opts.chartVersion
			if chartVersion == "" {
				return nil, fmt.Errorf(
					"missing required chart version argument. " +
						"use WithChartVersion() when passing an OCI " +
						"registry URL to Inspect().",
				)
			}
			untarDir, err := fetchOCI(ctx, subject, chartVersion, rc)
			if err != nil {
				return nil, err
			}
			defer os.RemoveAll(untarDir)
			// untarDir is a directory that contains a single directory with a
			// name equal to the Helm Chart...
			chartDir, err := firstDir(untarDir)
			if err != nil {
				return nil, err
			}
			hc, err = loader.Load(chartDir)
			if err != nil {
				return nil, err
			}
		} else if strings.HasPrefix(subject, "http") {
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
	return &Chart{
		Chart:       hc,
		inspectOpts: opts,
	}, nil
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

// fetchOCI pulls the a Helm Chart OCI artifact from the supplied OCI
// repository URL. It returns the local directory containing the pulled and
// untarred Helm Chart. Callers are responsible for cleaning up this directory.
func fetchOCI(
	ctx context.Context,
	repoURL string,
	chartVersion string,
	registryClient *registry.Client,
) (string, error) {
	ctx = debug.PushTrace(ctx, "helm:fetch-oci")
	defer debug.PopTrace(ctx)
	untarDir, err := os.MkdirTemp("", "kube-inspect-oci-pull")
	if err != nil {
		return "", fmt.Errorf("failed to create untar dir: %w", err)
	}
	debug.Printf(ctx, "created untar dir %s", untarDir)

	settings := cli.New()
	cfg := &action.Configuration{}
	err = cfg.Init(
		settings.RESTClientGetter(),
		settings.Namespace(),
		os.Getenv("HELM_DRIVER"),
		log.Printf,
	)
	if err != nil {
		return "", fmt.Errorf("failed to init action config: %w", err)
	}
	pull := action.NewPullWithOpts(action.WithConfig(cfg))
	pull.Settings = settings
	pull.Version = chartVersion
	pull.UntarDir = untarDir
	pull.Untar = true
	pull.SetRegistryClient(registryClient)

	_, err = pull.Run(repoURL)
	if err != nil {
		os.RemoveAll(untarDir)
		return "", fmt.Errorf("failed to pull chart: %w", err)
	}
	return untarDir, nil
}

// firstDir returns the path to the first (sub)directory in the supplied path.
func firstDir(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	if len(entries) != 1 {
		return "", fmt.Errorf(
			"expected single subdirectory but got %d.", len(entries),
		)
	}

	for _, v := range entries {
		if v.IsDir() {
			return filepath.Join(dir, v.Name()), nil
		}
	}
	return "", fmt.Errorf("single entry in directory was not a directory.")
}
