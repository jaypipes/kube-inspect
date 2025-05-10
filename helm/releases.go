// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package helm

import (
	"context"
	"os"

	"helm.sh/helm/v4/pkg/action"
	"helm.sh/helm/v4/pkg/cli"
	helmrelease "helm.sh/helm/v4/pkg/release/v1"

	"github.com/jaypipes/kube-inspect/debug"
)

const (
	envKeyHelmStorage  = "HELM_DRIVER"
	defaultHelmStorage = "secret"
)

// ListReleasesOptions is a mechanism for you to control the listing of Helm
// ListReleases.
type ListReleasesOptions struct {
	storage string
}

type ListReleasesOption func(opts *ListReleasesOptions)

// WithHelmStorage allows overriding the Helm Storage backend. If this option
// is not passed to Releases(), the HELM_DRIVER environment variable will be
// queried. If that isn't set, defaults to the "secret" Helm Storage backend.
func WithStorageBackend(backend string) ListReleasesOption {
	return func(opts *ListReleasesOptions) {
		opts.storage = backend
	}
}

// ListReleases returns a slice of Helm Release structs gathered from whatever
// the Helm storage driver is configured for the supplied Kubernetes cluster.
func ListReleases(
	ctx context.Context,
	opt ...ListReleasesOption,
) ([]*helmrelease.Release, error) {
	opts := &ListReleasesOptions{}
	for _, o := range opt {
		o(opts)
	}
	helmDriver := opts.storage
	if helmDriver == "" {
		helmDriver = os.Getenv(envKeyHelmStorage)
	}
	if helmDriver == "" {
		helmDriver = defaultHelmStorage
	}
	ctx = debug.PushTrace(ctx, "helm:releases")
	defer debug.PopTrace(ctx)
	var err error
	settings := cli.New()
	cfg := &action.Configuration{}
	err = cfg.Init(settings.RESTClientGetter(), "", helmDriver)
	if err != nil {
		return nil, err
	}
	lister := action.NewList(cfg)
	return lister.Run()
}
