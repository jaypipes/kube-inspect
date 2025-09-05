package helm

import "strings"

// ChartLocation describes where a HelmChart can be found.
type ChartLocation struct {
	// URL is the local filepath or OCI repository URL for the Helm Chart. If
	// empty, the Registry and ChartName fields should be non-empty.
	URL string `json:"url,omitempty"`
	// Registry is the Helm Chart registry URL for the Helm Chart. If empty,
	// URL should be non-empty and contain the OCI repository URL for the Helm
	// Chart.
	Registry string `json:"registry,omitempty"`
	// Name is the name for the Helm Chart. If empty, URL should be non-empty
	// and contain the OCI repository URL for the Helm Chart. If not empty,
	// Registry should also be not empty.
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

// IsHelmRegistry returns true if the ChartLocation refers to a Helm Chart
// Registry and Chart Name, false otherwise.
func (o *ChartLocation) IsHelmRegistry() bool {
	return o.URL == ""
}
