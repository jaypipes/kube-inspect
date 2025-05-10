// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package testutil

import (
	"os"
	"testing"
)

const (
	envKeySkipNetworkFetch = "SKIP_NETWORK_FETCH"
	envKeySkipKind         = "SKIP_KIND"
)

func SkipNetworkFetch(t *testing.T) {
	if _, ok := os.LookupEnv(envKeySkipNetworkFetch); ok {
		t.Skip("network fetching disabled.")
	}
}

func SkipIfNoKind(t *testing.T) {
	_, ok := os.LookupEnv(envKeySkipKind)
	if ok {
		t.Skipf("test requires KinD.")
	}
}
