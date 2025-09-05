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
			"oci://charts.jetstack.io/cert-manager",
			nil,
			false,
			true,
			false,
			&helm.ChartLocation{
				URL: "oci://charts.jetstack.io/cert-manager",
			},
		},
		{
			"Helm Registyr URL",
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
