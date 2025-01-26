package helm_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	helminspect "github.com/jaypipes/kube-inspect/helm"
)

const (
	nginxChartURL = "https://charts.bitnami.com/bitnami/nginx-8.8.4.tgz"
)

func TestInspectURL(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	c, err := helminspect.Inspect(nginxChartURL)
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
