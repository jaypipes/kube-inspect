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
		"admissionregistration.k8s.io/v1/MutatingWebhookConfiguration/cert-manager-webhook",
		"admissionregistration.k8s.io/v1/ValidatingWebhookConfiguration/cert-manager-webhook",
		"apps/v1/Deployment/cert-manager",
		"apps/v1/Deployment/cert-manager-cainjector",
		"apps/v1/Deployment/cert-manager-webhook",
		"rbac.authorization.k8s.io/v1/ClusterRole/cert-manager-cainjector",
		"rbac.authorization.k8s.io/v1/ClusterRole/cert-manager-cluster-view",
		"rbac.authorization.k8s.io/v1/ClusterRole/cert-manager-controller-approve:cert-manager-io",
		"rbac.authorization.k8s.io/v1/ClusterRole/cert-manager-controller-certificates",
		"rbac.authorization.k8s.io/v1/ClusterRole/cert-manager-controller-certificatesigningrequests",
		"rbac.authorization.k8s.io/v1/ClusterRole/cert-manager-controller-clusterissuers",
		"rbac.authorization.k8s.io/v1/ClusterRole/cert-manager-controller-ingress-shim",
		"rbac.authorization.k8s.io/v1/ClusterRole/cert-manager-controller-issuers",
		"rbac.authorization.k8s.io/v1/ClusterRole/cert-manager-controller-orders",
		"rbac.authorization.k8s.io/v1/ClusterRole/cert-manager-edit",
		"rbac.authorization.k8s.io/v1/ClusterRole/cert-manager-view",
		"rbac.authorization.k8s.io/v1/ClusterRole/cert-manager-webhook:subjectaccessreviews",
		"rbac.authorization.k8s.io/v1/ClusterRoleBinding/cert-manager-cainjector",
		"rbac.authorization.k8s.io/v1/ClusterRoleBinding/cert-manager-controller-approve:cert-manager-io",
		"rbac.authorization.k8s.io/v1/ClusterRoleBinding/cert-manager-controller-certificates",
		"rbac.authorization.k8s.io/v1/ClusterRoleBinding/cert-manager-controller-certificatesigningrequests",
		"rbac.authorization.k8s.io/v1/ClusterRoleBinding/cert-manager-controller-clusterissuers",
		"rbac.authorization.k8s.io/v1/ClusterRoleBinding/cert-manager-controller-ingress-shim",
		"rbac.authorization.k8s.io/v1/ClusterRoleBinding/cert-manager-controller-issuers",
		"rbac.authorization.k8s.io/v1/ClusterRoleBinding/cert-manager-controller-orders",
		"rbac.authorization.k8s.io/v1/ClusterRoleBinding/cert-manager-webhook:subjectaccessreviews",
		"rbac.authorization.k8s.io/v1/Role/cert-manager-cainjector:leaderelection",
		"rbac.authorization.k8s.io/v1/Role/cert-manager-tokenrequest",
		"rbac.authorization.k8s.io/v1/Role/cert-manager-webhook:dynamic-serving",
		"rbac.authorization.k8s.io/v1/Role/cert-manager:leaderelection",
		"rbac.authorization.k8s.io/v1/RoleBinding/cert-manager-cainjector:leaderelection",
		"rbac.authorization.k8s.io/v1/RoleBinding/cert-manager-kube-inspect-cert-manager-tokenrequest",
		"rbac.authorization.k8s.io/v1/RoleBinding/cert-manager-webhook:dynamic-serving",
		"rbac.authorization.k8s.io/v1/RoleBinding/cert-manager:leaderelection",
		"v1/Service/cert-manager",
		"v1/Service/cert-manager-cainjector",
		"v1/Service/cert-manager-webhook",
		"v1/ServiceAccount/cert-manager",
		"v1/ServiceAccount/cert-manager-cainjector",
		"v1/ServiceAccount/cert-manager-webhook",
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
