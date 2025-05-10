// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package helm_test

import (
	"testing"

	gdtcontext "github.com/gdt-dev/gdt/context"
	kindfix "github.com/gdt-dev/kube/fixtures/kind"
	kictx "github.com/jaypipes/kube-inspect/context"
	kihelm "github.com/jaypipes/kube-inspect/helm"
	"github.com/jaypipes/kube-inspect/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v4/pkg/action"
	"helm.sh/helm/v4/pkg/cli"
	helmkube "helm.sh/helm/v4/pkg/kube"
	"helm.sh/helm/v4/pkg/registry"
)

func TestReleasesSecret(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	ctx := kictx.New(kictx.WithDebug())
	ctx = gdtcontext.SetDebug(ctx)

	kind := kindfix.New(
		kindfix.WithDeleteOnStop(),
	)
	kind.Start(ctx)
	t.Cleanup(func() {
		kind.Stop(ctx)
	})

	chart, err := kihelm.Inspect(
		ctx, testutil.NginxLocalChartDir,
	)
	require.Nil(err)
	require.NotNil(chart.Chart)

	rels, err := kihelm.ListReleases(ctx)
	require.Nil(err)
	assert.Len(rels, 0)

	settings := cli.New()
	cfg := &action.Configuration{}
	err = cfg.Init(settings.RESTClientGetter(), settings.Namespace(), "secret")
	require.Nil(err)

	regClient, err := registry.NewClient()
	require.Nil(err)
	installer := action.NewInstall(cfg)
	installer.SetRegistryClient(regClient)
	installer.ReleaseName = "nginx"
	installer.Namespace = "test-nginx"
	installer.WaitStrategy = helmkube.HookOnlyStrategy

	r, err := installer.Run(chart.Chart, nil)
	require.Nil(err)
	assert.NotNil(r)

	rels, err = kihelm.ListReleases(ctx)
	require.Nil(err)
	assert.Len(rels, 1)
}
