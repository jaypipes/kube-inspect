// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package helm_test

import (
	"context"
	"fmt"
	"os"
	"slices"
	"testing"

	kihelm "github.com/jaypipes/kube-inspect/helm"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestChartDiff(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	ctx := context.TODO()

	af, err := os.Open(certManager1_17_1_LocalChartPath)
	require.Nil(err)
	ac, err := kihelm.Inspect(ctx, af)
	require.Nil(err)

	bf, err := os.Open(certManager1_18_0_LocalChartPath)
	require.Nil(err)
	bc, err := kihelm.Inspect(ctx, bf)
	require.Nil(err)

	diff, err := ac.Diff(ctx, bc)
	require.Nil(err)
	require.NotNil(diff)

	added := lo.Map(
		diff.Resources.Added,
		func(res *unstructured.Unstructured, _ int) string {
			return fmt.Sprintf("%s/%s", res.GetKind(), res.GetName())
		},
	)
	slices.Sort(added)
	expectAdded := []string{
		"ClusterRole/cert-manager-dns01-controller-challenges",
		"ClusterRole/cert-manager-http01-controller-challenges",
		"ClusterRoleBinding/cert-manager-dns01-controller-challenges",
		"ClusterRoleBinding/cert-manager-http01-controller-challenges",
	}
	assert.Equal(expectAdded, added)

	removed := lo.Map(
		diff.Resources.Removed,
		func(res *unstructured.Unstructured, _ int) string {
			return fmt.Sprintf("%s/%s", res.GetKind(), res.GetName())
		},
	)
	slices.Sort(removed)
	expectRemoved := []string{
		"ClusterRole/cert-manager-controller-challenges",
		"ClusterRoleBinding/cert-manager-controller-challenges",
	}
	assert.Equal(expectRemoved, removed)

	changed := lo.Keys(diff.Resources.Changed)
	slices.Sort(changed)
	expectChanged := []string{
		"cert-manager",
		"cert-manager-cainjector",
		"cert-manager-cainjector:leaderelection",
		"cert-manager-cluster-view",
		"cert-manager-controller-approve:cert-manager-io",
		"cert-manager-controller-certificates",
		"cert-manager-controller-certificatesigningrequests",
		"cert-manager-controller-clusterissuers",
		"cert-manager-controller-ingress-shim",
		"cert-manager-controller-issuers",
		"cert-manager-controller-orders",
		"cert-manager-edit",
		"cert-manager-kube-inspect-cert-manager-tokenrequest",
		"cert-manager-tokenrequest",
		"cert-manager-view",
		"cert-manager-webhook",
		"cert-manager-webhook:dynamic-serving",
		"cert-manager-webhook:subjectaccessreviews",
		"cert-manager:leaderelection",
	}
	assert.Equal(expectChanged, changed)

	assert.NotNil(diff.Values)
	require.Nil(err)
	expectValsDiff := `@@ global.rbac @@
! + one map entry added:
+   disableHTTPChallengesRole: false

@@ prometheus.servicemonitor.targetPort @@
! ± type change from int to string
- 9402
+ http-metrics`
	assert.Equal(expectValsDiff, diff.Values.String())
}
