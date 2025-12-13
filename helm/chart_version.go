package helm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/samber/lo"
	helmrepo "helm.sh/helm/v3/pkg/repo"
	"oras.land/oras-go/v2/content"
	ociremote "oras.land/oras-go/v2/registry/remote"
)

const (
	// DefaultChartVersionsLimit is the number of chart versions (OCI tags)
	// that we will returns by default if the ChartVersionsWithLimit() is not
	// set.
	DefaultChartVersionsLimit = 1000
)

// DefaultChartVersionsOptions is the default options we use when
// retrieving chart versions.
func defaultChartVersionsOptions() *ChartVersionsOptions {
	return &ChartVersionsOptions{
		limit:          DefaultChartVersionsLimit,
		errorCollector: io.Discard,
	}
}

// ChartVersion contains fields that describe a published version of a Helm
// Chart.
type ChartVersion struct {
	// Version is the SemVer2-compliant Helm Chart version -- i.e. the exact
	// value of the `Version` field in the `Chart.yaml` file. For Helm Charts
	// that are published in OCI registries, this will always correspond to the
	// tag for the Helm Chart OCI artifact.
	Version string
	// PublishedOn is the date that the ChartVersion was published, if known.
	PublishedOn string
	// Deprecated indicates whether the ChartVersion is deprecated.
	Deprecated bool
}

// ChartVersionFilter represents a filtering expression for ChartVersions
// returned by a call to `ChartVersionsFromLocation()`
type ChartVersionsFilter func(ver *ChartVersion, _ int) bool

// ChartVersionsOptions contains options for looking up chart versions
type ChartVersionsOptions struct {
	filters         []ChartVersionsFilter
	limit           int
	ociFetchDetails bool
	errorCollector  io.Writer
}

// ChartVersionsOption modifies the call to retrieve ChartVersions
type ChartVersionsOption func(*ChartVersionsOptions)

// ChartVersionsWithFilter returns a ChartVersionsOption that filters returned
// ChartVersions.
func ChartVersionsWithFilter(filter ChartVersionsFilter) ChartVersionsOption {
	return func(o *ChartVersionsOptions) {
		o.filters = append(o.filters, filter)
	}
}

// ChartVersionsWithOCIFetchDetails returns a ChartVersionsOption that enables
// fetching of published dates and deprecation information. Note: for Helm
// Charts published on OCI repositories, this dramatically increases the time
// to fetch chart version information. Don't blame kube-inspect, though. Blame
// the OCI distribution spec's terrible metadata handling queries.
func ChartVersionsWithOCIFetchDetails() ChartVersionsOption {
	return func(o *ChartVersionsOptions) {
		o.ociFetchDetails = true
	}
}

// ChartVersionsMatchingConstraint returns a ChartVersionOption that filters
// returned ChartVersions by the supplied SemVer constraint.
//
// Usage:
//
// | import (
// |	"context"
// |
// |	"github.com/jaypipes/kube-inspect/helm"
// | 	"github.com/MasterMinds/semver/v3"
// | )
// |
// | con, err := semver.NewConstraint(">1.2")
// | if err != nil {
// |	panic(err)
// | }
// |
// | loc, err := helm.ChartLocationFromURL(
// |	"oci://quay.io/jetstack/charts/cert-manager",
// | )
// | if err != nil {
// |	panic(err)
// | }
// | vers, err := helm.ChartVersionsFromLocation(
// |	context.Background(),
// |	loc,
// |	helm.ChartVersionsMatchingConstraint(*con),
// | }
func ChartVersionsMatchingConstraint(con semver.Constraints) ChartVersionsOption {
	return func(o *ChartVersionsOptions) {
		filter := func(ver *ChartVersion, _ int) bool {
			sv, err := semver.StrictNewVersion(ver.Version)
			if err != nil {
				return false
			}
			return con.Check(sv)
		}
		o.filters = append(o.filters, filter)
	}
}

// ChartVersionsWithLimit returns a ChartVersionOption that limits the number
// of ChartVersions returned. The limit is applied to ChartVersions that match
// all supplied filters.
func ChartVersionsWithLimit(limit int) ChartVersionsOption {
	return func(o *ChartVersionsOptions) {
		o.limit = limit
	}
}

// ChartVersionsWithErrorCollector returns a ChartVersionOption that writes any
// errors found during retrieval of chart versions to the supplied io.Writer.
func ChartVersionsWithErrorCollector(w io.Writer) ChartVersionsOption {
	return func(o *ChartVersionsOptions) {
		o.errorCollector = w
	}
}

// ChartVersionsFromLocation returns a slice of `ChartVersion`structs queried
// from the OCI or Helm Repository associated with the supplied ChartLocation.
func ChartVersionsFromLocation(
	ctx context.Context,
	loc *ChartLocation,
	opt ...ChartVersionsOption,
) ([]*ChartVersion, error) {
	if loc.IsOCI() {
		repo, err := loc.OCIRepository()
		if err != nil {
			return nil, err
		}
		return ChartVersionsFromOCIRepository(ctx, repo, opt...)
	} else if loc.IsHelmRepository() {
		repo, err := loc.HelmRepository()
		if err != nil {
			return nil, err
		}
		return ChartVersionsFromHelmRepository(
			ctx, repo, loc.Name, opt...,
		)
	}
	return nil, fmt.Errorf(
		"unable to find chart versions from ChartLocation",
	)
}

// ChartVersionsFromOCIRepository returns a slice of `ChartVersion` structs
// queried from the supplied OCI Repository.
func ChartVersionsFromOCIRepository(
	ctx context.Context,
	repo *ociremote.Repository,
	opt ...ChartVersionsOption,
) ([]*ChartVersion, error) {
	opts := defaultChartVersionsOptions()
	for _, o := range opt {
		o(opts)
	}
	out := []*ChartVersion{}
	matched := 0
	// We need to track matched tags because sometimes chart authors publish
	// tags that have the same version with and without a "v" prefix :(
	seenTags := []string{}
	err := repo.Tags(ctx, "", func(tags []string) error {
		for _, tag := range tags {
			if matched >= opts.limit {
				break
			}
			// We need to filter out non-chart tags (like SBOMs and
			// signatures). The most accurate way of doing this is using the
			// semver.StrictNewVersion() since Helm Chart Versions are required
			// to be valid strict SemVer2-compliant. However, we see in the
			// wild authors using the "v" prefix erroneously, so here we
			// automatically trim a "v" if it's the first character of the tag
			// and use the semver.StrictNewVersion() function on the stripped
			// string.
			tag = strings.TrimPrefix(tag, "v")
			_, err := semver.StrictNewVersion(tag)
			if err != nil {
				msg := fmt.Sprintf(
					"version %q was not valid semver\n", tag,
				)
				opts.errorCollector.Write([]byte(msg)) // nolint:errcheck
				continue
			}
			cv := &ChartVersion{Version: tag}
			exclude := false
			for _, filter := range opts.filters {
				if !filter(cv, 0) {
					exclude = true
					break
				}
			}
			if exclude || lo.Contains(seenTags, tag) {
				continue
			}
			matched++
			seenTags = append(seenTags, tag)
			out = append(out, cv)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if opts.ociFetchDetails {
		// Now we need to get the published on dates by examining the OCI manifests
		// associated with each matched version tag. sigh, I hate the OCI metadata
		// retrieval APIs and how they force you into inefficient N+1 queries :(
		for x, cv := range out {
			desc, rc, err := repo.FetchReference(ctx, cv.Version)
			if err != nil {
				msg := fmt.Sprintf(
					"failed to fetch reference for version %q: %s\n",
					cv.Version, err,
				)
				opts.errorCollector.Write([]byte(msg)) // nolint:errcheck
				continue
			}
			defer rc.Close()
			manifestBytes, err := content.ReadAll(rc, desc)
			if err != nil {
				msg := fmt.Sprintf(
					"failed to read OCI descriptor for version %q: %s\n",
					cv.Version, err,
				)
				opts.errorCollector.Write([]byte(msg)) // nolint:errcheck
				continue
			}

			var manifest ocispec.Manifest
			if err = json.Unmarshal(manifestBytes, &manifest); err != nil {
				msg := fmt.Sprintf(
					"failed to unmarshal version %q into manifest: %s\n"+
						"manifest bytes: %s\n",
					cv.Version, err, string(manifestBytes),
				)
				opts.errorCollector.Write([]byte(msg)) // nolint:errcheck
				continue
			}

			for k, v := range manifest.Annotations {
				if k == "org.opencontainers.image.created" {
					publishedOn, err := time.Parse(time.RFC3339, v)
					if err != nil {
						msg := fmt.Sprintf(
							"failed to parse org.opencontainers.image.created "+
								"of %q for version %q: %s\n",
							v, cv.Version, err,
						)
						opts.errorCollector.Write([]byte(msg)) // nolint:errcheck
						continue
					}
					cv.PublishedOn = publishedOn.Format(time.DateTime)
					out[x] = cv
					break
				}
			}
		}
	}
	return out, nil
}

// ChartVersionsFromHelmRepository returns a slice of `ChartVersion` structs
// queried from the supplied Helm Repository and chart name.
func ChartVersionsFromHelmRepository(
	ctx context.Context,
	repo *helmrepo.ChartRepository,
	chartName string,
	opt ...ChartVersionsOption,
) ([]*ChartVersion, error) {
	opts := defaultChartVersionsOptions()
	for _, o := range opt {
		o(opts)
	}
	indexPath, err := repo.DownloadIndexFile()
	if err != nil {
		return nil, err
	}

	indexFile, err := helmrepo.LoadIndexFile(indexPath)
	if err != nil {
		return nil, err
	}
	matched := 0
	out := []*ChartVersion{}
	for _, cv := range indexFile.Entries[chartName] {
		if matched >= opts.limit {
			break
		}
		// Yep, Helm Repositories regularly publish non-compliant SemVer2 chart
		// versions, so we need to be lenient here and auto-trim the "v" prefix
		// while checking for valid chart versions.
		ver := strings.TrimPrefix(cv.Version, "v")
		_, err := semver.StrictNewVersion(ver)
		if err != nil {
			msg := fmt.Sprintf(
				"version %q was not valid semver.", ver,
			)
			opts.errorCollector.Write([]byte(msg)) // nolint:errcheck
			continue
		}
		var publishedOn string
		if !cv.Created.IsZero() {
			publishedOn = cv.Created.Format(time.DateTime)
		}
		cv := &ChartVersion{
			Version:     ver,
			PublishedOn: publishedOn,
			Deprecated:  cv.Deprecated,
		}
		exclude := false
		for _, filter := range opts.filters {
			if !filter(cv, 0) {
				exclude = true
				break
			}
		}
		if exclude {
			continue
		}
		matched++
		out = append(out, cv)
	}
	return out, nil
}
