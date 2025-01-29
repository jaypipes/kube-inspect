// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package helm_test

import (
	"context"
	"testing"

	kihelm "github.com/jaypipes/kube-inspect/helm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOptionalResources(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	ctx := context.TODO()
	c, err := kihelm.Inspect(
		ctx, nginxLocalChartDir,
	)
	require.Nil(err)

	require.NotNil(c.Metadata)
	assert.Equal("nginx", c.Metadata.Name)
	optResources, err := c.OptionalResources(ctx)
	require.Nil(err)
	toggles := []string{}
	for _, r := range optResources {
		toggles = append(toggles, r.ValueToggle)
	}
	// NOTE(jaypipes): The nginx Helm Chart in testdata contains only a partial
	// JSONSchema that does not fully describe the entire values.yaml
	// collection of configuration values. Notably, the "serviceAccount.create"
	// property is missing (but the "pdb.create" property is present in the
	// schema)
	assert.Contains(toggles, "pdb.create")
}
