// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package helm_test

import (
	"context"
	"os"
	"strings"
	"testing"

	kictx "github.com/jaypipes/kube-inspect/context"
	kihelm "github.com/jaypipes/kube-inspect/helm"
	"github.com/jaypipes/kube-inspect/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v4/pkg/chart/v2/loader"
)

func TestInspectURL(t *testing.T) {
	testutil.SkipNetworkFetch(t)
	require := require.New(t)
	assert := assert.New(t)
	ctx := kictx.New(kictx.WithDebug())
	c, err := kihelm.Inspect(ctx, testutil.NginxChartURL)
	require.Nil(err)

	require.NotNil(c.Metadata)
	assert.Equal("nginx", c.Metadata.Name)
	resources, err := c.Resources(ctx)
	require.Nil(err)
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
	tf, err := os.Open(testutil.NginxLocalChartPath)
	require.Nil(err)
	hc, err := loader.LoadArchive(tf)
	require.Nil(err)
	ctx := context.TODO()
	c, err := kihelm.Inspect(ctx, hc)
	require.Nil(err)

	require.NotNil(c.Metadata)
	assert.Equal("nginx", c.Metadata.Name)
	resources, err := c.Resources(ctx)
	require.Nil(err)
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
	ctx := context.TODO()
	c, err := kihelm.Inspect(ctx, testutil.NginxLocalChartDir)
	require.Nil(err)

	require.NotNil(c.Metadata)
	assert.Equal("nginx", c.Metadata.Name)
	resources, err := c.Resources(ctx)
	require.Nil(err)
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
	tf, err := os.Open(testutil.NginxLocalChartPath)
	require.Nil(err)
	ctx := context.TODO()
	c, err := kihelm.Inspect(ctx, tf)
	require.Nil(err)

	require.NotNil(c.Metadata)
	assert.Equal("nginx", c.Metadata.Name)
	resources, err := c.Resources(ctx)
	require.Nil(err)
	resourceKinds := []string{}
	for _, r := range resources {
		resourceKinds = append(resourceKinds, r.GetKind())
	}
	assert.Len(resources, 3)
	assert.Contains(resourceKinds, "Deployment")
	assert.Contains(resourceKinds, "Service")
	assert.Contains(resourceKinds, "ConfigMap")
}

func TestInspectWithValues(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	overrides := "pdb.create=true,serviceAccount.create=true"
	ctx := context.TODO()
	c, err := kihelm.Inspect(
		ctx, testutil.NginxLocalChartDir,
		kihelm.WithValues(overrides),
	)
	require.Nil(err)

	require.NotNil(c.Metadata)
	assert.Equal("nginx", c.Metadata.Name)
	resources, err := c.Resources(ctx)
	require.Nil(err)
	resourceKinds := []string{}
	for _, r := range resources {
		resourceKinds = append(resourceKinds, r.GetKind())
	}
	assert.Len(resources, 5)
	assert.Contains(resourceKinds, "ServiceAccount")
	assert.Contains(resourceKinds, "PodDisruptionBudget")
	assert.Contains(resourceKinds, "Deployment")
	assert.Contains(resourceKinds, "Service")
	assert.Contains(resourceKinds, "ConfigMap")
}

// When a Helm Chart specifies a KubeVersion constraint that does not meet the
// "DefaultCapabilities.KubeVersion" set in the Helm SDK Go's chartutil
// package, we need to detect that and automatically adjust the
// installer.KubeVersion used in rendering.
//
// See: https://github.com/jaypipes/kube-inspect/issues/2
// See: https://github.com/helm/helm/blob/3a94215585b91d5ac41ebb258e376aa11980b564/pkg/chartutil/capabilities.go#L31-L50
func TestInspectChartAutoAdjustedKubeVersion(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	debugCollector := &strings.Builder{}
	tf, err := os.Open(testutil.CertManagerLocalChartPath)
	require.Nil(err)
	hc, err := loader.LoadArchive(tf)
	require.Nil(err)
	ctx := kictx.New(kictx.WithDebug(debugCollector))
	c, err := kihelm.Inspect(ctx, hc)
	require.Nil(err)

	require.NotNil(c.Metadata)
	assert.Equal("cert-manager", c.Metadata.Name)
	_, err = c.Resources(ctx)

	require.Nil(err)
	debugContent := debugCollector.String()
	expected := `version check failed for default Helm SDK ` +
		`kubeVersion "1.20.0". setting installer.KubeVersion ` +
		`manually to "v1.22.0-0"`
	assert.Contains(debugContent, expected)
}
