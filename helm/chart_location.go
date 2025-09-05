package helm

import (
	"fmt"
	"strings"

	"helm.sh/helm/v3/pkg/registry"
)

// ChartLocation describes where a HelmChart can be found.
type ChartLocation struct {
	// URL is the local filepath or OCI repository URL for the Helm Chart. If
	// empty, the Repository and ChartName fields should be non-empty.
	URL string `json:"url,omitempty"`
	// Repository is the Helm Chart repository URL for the Helm Chart. If empty,
	// URL should be non-empty and contain the OCI repository URL for the Helm
	// Chart.
	Repository string `json:"repository,omitempty"`
	// Name is the name for the Helm Chart. If empty, URL should be non-empty
	// and contain the OCI repository URL for the Helm Chart. If not empty,
	// Repository should also be not empty.
	Name string `json:"name,omitempty"`
}

// IsLocal returns true if the ChartLocation refers to a local filesystem path,
// false otherwise.
func (o *ChartLocation) IsLocal() bool {
	return strings.HasPrefix(o.URL, "file://")
}

// IsOCI returns true if the ChartLocation refers to an OCI repository URL,
// false otherwise.
func (o *ChartLocation) IsOCI() bool {
	return strings.HasPrefix(o.URL, "oci://")
}

// IsHelmRepository returns true if the ChartLocation refers to a Helm Chart
// Registry and Chart Name, false otherwise.
func (o *ChartLocation) IsHelmRepository() bool {
	return o.URL == ""
}

// ChartLocationFromURL returns a ChartLocation given a supplied URL. If the
// supplied URL is an HTTP(S) URL, it is expected to be in the format
// http(s)://<helm repository>/<chart_name>.
func ChartLocationFromURL(url string) (*ChartLocation, error) {
	if registry.IsOCI(url) || strings.HasPrefix(url, "file://") {
		return &ChartLocation{URL: url}, nil
	}
	if !strings.HasPrefix(url, "https://") &&
		!strings.HasPrefix(url, "http://") {
		return nil, fmt.Errorf(
			"invalid URL format, expected <helm repository>/<chart>: %s",
			url,
		)
	}
	parts := strings.Split(url, "/")
	if len(parts) < 2 {
		return nil, fmt.Errorf(
			"invalid URL format, expected <helm repository>/<chart>: %s",
			url,
		)
	}

	repo := strings.Join(parts[:len(parts)-1], "/")
	chartName := parts[len(parts)-1]
	if repo == "" || chartName == "" {
		return nil, fmt.Errorf(
			"invalid URL format, expected <helm repository>/<chart>: %s",
			url,
		)
	}
	return &ChartLocation{Repository: repo, Name: chartName}, nil
}
