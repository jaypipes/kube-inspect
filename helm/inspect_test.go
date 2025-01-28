// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package helm_test

import (
	"testing"

	kictx "github.com/jaypipes/kube-inspect/context"
	kihelm "github.com/jaypipes/kube-inspect/helm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	nginxChartURL = "https://charts.bitnami.com/bitnami/nginx-8.8.4.tgz"
)

func TestInspectURL(t *testing.T) {
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
