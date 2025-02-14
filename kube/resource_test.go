// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package kube_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jaypipes/kube-inspect/kube"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	singleDeploymentManifest    = filepath.Join("testdata", "deployment.yaml")
	multipleDeploymentsManifest = filepath.Join("testdata", "deployments.yaml")
	emptyDocumentManifest       = filepath.Join("testdata", "empty.yaml")
)

func TestResourcesFromManifest_SingleDeployment(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	ctx := context.TODO()
	contents, err := os.ReadFile(singleDeploymentManifest)
	require.Nil(err)

	b := bytes.NewBuffer(contents)
	resources, err := kube.ResourcesFromManifest(ctx, b)
	require.Nil(err)
	require.NotNil(resources)

	resourceKinds := []string{}
	for _, r := range resources {
		resourceKinds = append(resourceKinds, r.GetKind())
	}
	resourceKinds = lo.Uniq(resourceKinds)
	assert.Len(resources, 1)
	assert.Contains(resourceKinds, "Deployment")
	assert.Equal("nginx-deployment", resources[0].GetName())
}

func TestResourcesFromManifest_MultipleDeployments(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	ctx := context.TODO()
	contents, err := os.ReadFile(multipleDeploymentsManifest)
	require.Nil(err)

	b := bytes.NewBuffer(contents)
	resources, err := kube.ResourcesFromManifest(ctx, b)
	require.Nil(err)
	require.NotNil(resources)

	resourceKinds := []string{}
	for _, r := range resources {
		resourceKinds = append(resourceKinds, r.GetKind())
	}
	resourceKinds = lo.Uniq(resourceKinds)
	assert.Len(resources, 3)
	assert.Len(resourceKinds, 1)
	assert.Contains(resourceKinds, "Deployment")
	assert.Equal("nginx-deployment1", resources[0].GetName())
	assert.Equal("nginx-deployment2", resources[1].GetName())
	assert.Equal("nginx-deployment3", resources[2].GetName())
}

func TestResourcesFromManifest_EmptyDocument(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	ctx := context.TODO()
	contents, err := os.ReadFile(emptyDocumentManifest)
	require.Nil(err)

	b := bytes.NewBuffer(contents)
	resources, err := kube.ResourcesFromManifest(ctx, b)
	require.Nil(err)
	require.NotNil(resources)

	resourceKinds := []string{}
	for _, r := range resources {
		resourceKinds = append(resourceKinds, r.GetKind())
	}
	resourceKinds = lo.Uniq(resourceKinds)
	assert.Len(resources, 1)
	assert.Contains(resourceKinds, "Deployment")
}
