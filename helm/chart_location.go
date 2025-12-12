package helm

import (
	"fmt"
	"strings"

	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/repo"
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

// OCIRegistry returns the registry part of the OCI URL, or an empty string if
// the ChartLocation does not refer to an OCI Artifact.
//
// The returned string is normalized to not include any oci:// prefix and does
// not include a trailing slash. Therefore, if the ChartLocation.URL is
// `oci://quay.io/jetstack/charts/cert-manager`, the returned string
// will be `quay.io`.
func (o *ChartLocation) OCIRegistry() string {
	if !o.IsOCI() {
		return ""
	}
	url := strings.TrimPrefix(o.URL, "oci://")
	url = strings.TrimSuffix(url, "/")

	parts := strings.Split(url, "/")
	return parts[0]
}

// OCIRegistryAndNamespace returns the registry and namespace part of the OCI
// URL, or empty strings if the ChartLocation does not refer to an OCI
// Artifact.
//
// The returned strings are normalized to not include any oci:// prefix and
// to not include a trailing slash. Therefore, if the ChartLocation.URL is
// `oci://quay.io/jetstack/charts/cert-manager/`, the returned strings
// will be `quay.io` and `jetstack/charts`.
func (o *ChartLocation) OCIRegistryAndNamespace() (string, string) {
	if !o.IsOCI() {
		return "", ""
	}
	url := strings.TrimPrefix(o.URL, "oci://")
	url = strings.TrimSuffix(url, "/")

	parts := strings.Split(url, "/")
	if len(parts) < 2 {
		return "", ""
	}
	return parts[0], strings.Join(parts[1:len(parts)-1], "/")
}

// OCIRepository returns the ChartLocation's URL stripped of the `oci://`
// prefix.  This string is meant to be passed as-is to the
// `oras.land/oras-go/v2/registry/remote.NewRepository` function.
//
// Returns an empty string if the ChartLocation does not refer to an OCI
// Artifact or is malformed.
func (o *ChartLocation) OCIRepository() string {
	if !o.IsOCI() {
		return ""
	}
	url := strings.TrimPrefix(o.URL, "oci://")
	url = strings.TrimSuffix(url, "/")
	return url
}

// IsHelmRepository returns true if the ChartLocation refers to a Helm Chart
// Registry and Chart Name, false otherwise.
func (o *ChartLocation) IsHelmRepository() bool {
	return o.URL == ""
}

// HelmRepositoryURL returns a string with the well-formed Helm repository URL
// (including http(s):// prefix and no trailing slash). This URL does *not*
// include the Helm Chart name. Returns an empty string if the ChartLocation
// refers to an OCI Artifact or local file reference.
func (o *ChartLocation) HelmRepositoryURL() string {
	if o.IsOCI() || o.IsLocal() {
		return ""
	}
	return strings.TrimSuffix(o.Repository, "/")
}

// HelmRepository returns a `helm.sh/helm/v3/pkg/repo.Repo` object referring to
// the ChartLocation, or nil if the ChartLocation does not refer to a Helm
// Repository.
//
// This is a helper method to avoid going through all the helm.sh/helm/v3 SDK
// rigamorole around cli, settings, and getters.
func (o *ChartLocation) HelmRepository() (*repo.ChartRepository, error) {
	if !o.IsHelmRepository() {
		return nil, fmt.Errorf(
			"ChartLocation does not refer to a Helm Repository.",
		)
	}

	entry := &repo.Entry{
		Name: o.Name,
		URL:  o.HelmRepositoryURL(),
	}

	settings := cli.New()
	return repo.NewChartRepository(entry, getter.All(settings))
}

// ChartLocationFromURL returns a ChartLocation given a supplied URL. If the
// supplied URL is an HTTP(S) URL, it is expected to be in the format
// http(s)://<helm repository>/<chart_name>.
func ChartLocationFromURL(url string) (*ChartLocation, error) {
	if strings.HasPrefix(url, "file://") {
		return &ChartLocation{URL: url}, nil
	}
	if registry.IsOCI(url) {
		parts := strings.Split(url, "/")
		if len(parts) < 3 {
			return nil, fmt.Errorf(
				"invalid URL format, "+
					"expected oci://<registry>/<namespace>/<tag>: %s",
				url,
			)
		}
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
