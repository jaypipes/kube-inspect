// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package helm_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/jaypipes/kube-inspect/helm"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChartVersionsFromLocation(t *testing.T) {
	tcs := []struct {
		name                 string
		url                  string
		limit                int
		verConstraint        string
		ociFetchDetails      bool
		expLen               int
		expErr               error
		expContainsVersions  []string
		errCollector         *strings.Builder
		errCollectorContains string
	}{
		{
			// NOTE(jaypipes): this is likely a fragile test case because OCI
			// tag listing doesn't have any deterministic order :(
			"jetstack-cert-manager OCI lists 1.18.3",
			"oci://quay.io/jetstack/charts/cert-manager",
			100,
			"",
			false,
			100,
			nil,
			[]string{"1.18.3"},
			&strings.Builder{},
			"was not valid semver", // SBOM sigs aren't valid tags...
		},
		{
			"jetstack-cert-manager OCI with version constraint",
			"oci://quay.io/jetstack/charts/cert-manager",
			5,
			">=1.18.3, <=1.18.4",
			false,
			2,
			nil,
			[]string{"1.18.3", "1.18.4"},
			nil,
			"",
		},
		{
			"jetstack-cert-manager OCI with version constraint and OCI details",
			"oci://quay.io/jetstack/charts/cert-manager",
			5,
			"1.18.3",
			true,
			1,
			nil,
			[]string{"1.18.3"},
			nil,
			"",
		},
		{
			"jetstack helm repository with version constraint",
			"https://charts.jetstack.io/cert-manager",
			100,
			">=1.18.3, <=1.18.4",
			false,
			2,
			nil,
			[]string{"1.18.3", "1.18.4"},
			nil,
			"",
		},
	}
	ctx := context.TODO()
	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			assert := assert.New(tt)
			require := require.New(tt)
			opts := []helm.ChartVersionsOption{}
			if tc.limit > 0 {
				opts = append(opts, helm.ChartVersionsWithLimit(tc.limit))
			}
			if tc.verConstraint != "" {
				vc, err := semver.NewConstraint(tc.verConstraint)
				require.Nil(err)
				opts = append(opts, helm.ChartVersionsMatchingConstraint(*vc))
			}
			if tc.ociFetchDetails {
				opts = append(opts, helm.ChartVersionsWithOCIFetchDetails())
			}
			if tc.errCollector != nil {
				opts = append(
					opts,
					helm.ChartVersionsWithErrorCollector(
						tc.errCollector,
					),
				)
			}
			loc, err := helm.ChartLocationFromURL(tc.url)
			require.Nil(err)
			got, err := helm.ChartVersionsFromLocation(ctx, loc, opts...)
			if tc.expErr != nil {
				assert.Error(tc.expErr, err)
			} else {
				if tc.expContainsVersions != nil {
					vers := lo.Map(
						got, func(cv *helm.ChartVersion, _ int) string {
							return cv.Version
						},
					)
					assert.Len(got, tc.expLen, fmt.Sprintf("%v", vers))
					for _, search := range tc.expContainsVersions {
						assert.Contains(vers, search)
					}
				}
				if tc.ociFetchDetails {
					for _, cv := range got {
						assert.NotEmpty(cv.PublishedOn)
					}
				}
			}
			if tc.errCollector != nil {
				if tc.errCollectorContains != "" {
					errStr := tc.errCollector.String()
					assert.Contains(errStr, tc.errCollectorContains)
				}
			}
		})
	}
}
