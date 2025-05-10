// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package testutil

import (
	"path/filepath"
)

const (
	NginxChartURL = "https://charts.bitnami.com/bitnami/nginx-8.8.4.tgz"
)

var (
	CertManagerLocalChartPath = filepath.Join("testdata", "cert-manager-v1.17.1.tgz")
	NginxLocalChartPath       = filepath.Join("testdata", "nginx-8.8.4.tgz")
	NginxLocalChartDir        = filepath.Join("testdata", "nginx")
)
