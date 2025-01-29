// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package helm_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	kictx "github.com/jaypipes/kube-inspect/context"
	kihelm "github.com/jaypipes/kube-inspect/helm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chart/loader"
)

const (
	skipNetworkFetchEnvKey = "SKIP_NETWORK_FETCH"
	nginxChartURL          = "https://charts.bitnami.com/bitnami/nginx-8.8.4.tgz"
)

var (
	nginxLocalChartPath = filepath.Join("testdata", "nginx-8.8.4.tgz")
	nginxLocalChartDir  = filepath.Join("testdata", "nginx")
)

func skipNetworkFetch(t *testing.T) {
	if _, ok := os.LookupEnv(skipNetworkFetchEnvKey); ok {
		t.Skip("network fetching disabled.")
	}
}

func TestInspectURL(t *testing.T) {
	skipNetworkFetch(t)
	require := require.New(t)
	assert := assert.New(t)
	ctx := kictx.New(kictx.WithDebug())
	c, err := kihelm.Inspect(ctx, nginxChartURL)
	require.Nil(err)

	require.NotNil(c.Metadata)
	assert.Equal("nginx", c.Metadata.Name)
	resources := c.Resources()
	resourceKinds := []string{}
	for _, r := range resources {
		resourceKinds = append(resourceKinds, r.GetKind())
	}
	assert.Len(resources, 3)
	assert.Contains(resourceKinds, "Deployment")
	assert.Contains(resourceKinds, "Service")
	assert.Contains(resourceKinds, "ConfigMap")
}

func TestInspectHelmSDKChart(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	tf, err := os.Open(nginxLocalChartPath)
	require.Nil(err)
	hc, err := loader.LoadArchive(tf)
	require.Nil(err)
	c, err := kihelm.Inspect(context.TODO(), hc)
	require.Nil(err)

	require.NotNil(c.Metadata)
	assert.Equal("nginx", c.Metadata.Name)
	resources := c.Resources()
	resourceKinds := []string{}
	for _, r := range resources {
		resourceKinds = append(resourceKinds, r.GetKind())
	}
	assert.Len(resources, 3)
	assert.Contains(resourceKinds, "Deployment")
	assert.Contains(resourceKinds, "Service")
	assert.Contains(resourceKinds, "ConfigMap")
}

func TestInspectChartDir(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	c, err := kihelm.Inspect(context.TODO(), nginxLocalChartDir)
	require.Nil(err)

	require.NotNil(c.Metadata)
	assert.Equal("nginx", c.Metadata.Name)
	resources := c.Resources()
	resourceKinds := []string{}
	for _, r := range resources {
		resourceKinds = append(resourceKinds, r.GetKind())
	}
	assert.Len(resources, 3)
	assert.Contains(resourceKinds, "Deployment")
	assert.Contains(resourceKinds, "Service")
	assert.Contains(resourceKinds, "ConfigMap")
}

func TestInspectIOReader(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	tf, err := os.Open(nginxLocalChartPath)
	require.Nil(err)
	c, err := kihelm.Inspect(context.TODO(), tf)
	require.Nil(err)

	require.NotNil(c.Metadata)
	assert.Equal("nginx", c.Metadata.Name)
	resources := c.Resources()
	resourceKinds := []string{}
	for _, r := range resources {
		resourceKinds = append(resourceKinds, r.GetKind())
	}
	assert.Len(resources, 3)
	assert.Contains(resourceKinds, "Deployment")
	assert.Contains(resourceKinds, "Service")
	assert.Contains(resourceKinds, "ConfigMap")
}
