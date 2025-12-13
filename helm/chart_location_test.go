// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package helm_test

import (
	"fmt"
	"testing"

	"github.com/jaypipes/kube-inspect/helm"
	"github.com/stretchr/testify/assert"
)

func TestChartLocation(t *testing.T) {
	tcs := []struct {
		name             string
		url              string
		expErr           error
		isLocal          bool
		isOCI            bool
		isHelmRepository bool
		exp              *helm.ChartLocation
	}{
		{
			"local file path",
			"file://path/to/chart.tar.gz",
			nil,
			true,
			false,
			false,
			&helm.ChartLocation{
				URL: "file://path/to/chart.tar.gz",
			},
		},
		{
			"OCI URL",
			"oci://quay.io/jetstack/charts/cert-manager",
			nil,
			false,
			true,
			false,
			&helm.ChartLocation{
				URL: "oci://quay.io/jetstack/charts/cert-manager",
			},
		},
		{
			"Helm Registry URL",
			"https://charts.jetstack.io/cert-manager",
			nil,
			false,
			false,
			true,
			&helm.ChartLocation{
				Repository: "https://charts.jetstack.io",
				Name:       "cert-manager",
			},
		},
		{
			"error: no http(s) prefix",
			"charts.jetstack.io/cert-manager",
			fmt.Errorf(`invalid URL format, expected <helm repository>/<chart>: "charts.jetstack.io/cert-manager"`),
			false,
			false,
			false,
			nil,
		},
		{
			"error: not enough parts",
			"https://charts.jetstack.io",
			fmt.Errorf(`invalid URL format, expected <helm repository>/<chart>: "charts.jetstack.io"`),
			false,
			false,
			false,
			nil,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			assert := assert.New(tt)
			got, err := helm.ChartLocationFromURL(tc.url)
			if tc.expErr != nil {
				assert.Error(tc.expErr, err)
			} else {
				assert.Equal(tc.isLocal, got.IsLocal())
				assert.Equal(tc.isOCI, got.IsOCI())
				assert.Equal(tc.isHelmRepository, got.IsHelmRepository())
				assert.Equal(tc.exp, got)
			}
		})
	}
}

func TestChartLocationOCI(t *testing.T) {
	tcs := []struct {
		name              string
		url               string
		expRegistry       string
		expNamespace      string
		expRepositoryPath string
	}{
		{
			"single-part namespace",
			"oci://quay.io/bitnami-charts/cert-manager",
			"quay.io",
			"bitnami-charts",
			"quay.io/bitnami-charts/cert-manager",
		},
		{
			"dual-part namespace",
			"oci://quay.io/jetstack/charts/cert-manager",
			"quay.io",
			"jetstack/charts",
			"quay.io/jetstack/charts/cert-manager",
		},
		{
			"triple-part namespace",
			"oci://quay.io/jetstack/charts/gold/cert-manager",
			"quay.io",
			"jetstack/charts/gold",
			"quay.io/jetstack/charts/gold/cert-manager",
		},
		{
			"not OCI",
			"file://path/to/chart.tar.gz",
			"",
			"",
			"",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			assert := assert.New(tt)
			loc, err := helm.ChartLocationFromURL(tc.url)
			assert.Nil(err)
			reg := loc.OCIRegistry()
			assert.Equal(tc.expRegistry, reg)
			reg, ns := loc.OCIRegistryAndNamespace()
			assert.Equal(tc.expRegistry, reg)
			assert.Equal(tc.expNamespace, ns)
			repo := loc.OCIRepositoryPath()
			assert.Equal(tc.expRepositoryPath, repo)
		})
	}
}

func TestChartLocationHelmRepository(t *testing.T) {
	tcs := []struct {
		name             string
		url              string
		expRepositoryURL string
		expName          string
		expErr           error
	}{
		{
			"simple helm repository",
			"https://helm.sh/charts/cert-manager",
			"https://helm.sh/charts",
			"cert-manager",
			nil,
		},
		{
			"not Helm Repository",
			"oci://quay.io/jetstack/charts/gold/cert-manager",
			"",
			"",
			fmt.Errorf("ChartLocation does not refer to a Helm Repository."),
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			assert := assert.New(tt)
			loc, err := helm.ChartLocationFromURL(tc.url)
			assert.Nil(err)
			repoURL := loc.HelmRepositoryURL()
			assert.Equal(tc.expRepositoryURL, repoURL)
			repo, err := loc.HelmRepository()
			assert.Equal(tc.expName, loc.Name)
			if tc.expErr != nil {
				assert.Error(tc.expErr, err)
			} else {
				assert.NotNil(repo)
			}
		})
	}
}
