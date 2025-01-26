package helm

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"helm.sh/helm/v3/pkg/action"
	helmchart "helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
)

// Inspect returns a `Chart` that describes a Helm Chart that has been rendered
// to actual Kubernetes resource manifests.
//
// The `subject` argument can be a filepath, a URL, a helm sdk-go `*Chart`
// struct, or an `io.Reader` pointing at either a directory or a compressed tar
// archive.
func Inspect(subject any) (*Chart, error) {
	var err error
	var hc *helmchart.Chart
	switch subject := subject.(type) {
	case string:
		if strings.HasPrefix(subject, "http") {
			tf, err := fetchArchive(subject)
			if err != nil {
				return nil, err
			}
			defer os.Remove(tf.Name())
			hc, err = loader.LoadArchive(tf)
			if err != nil {
				return nil, fmt.Errorf("error loading archive: %w", err)
			}
		} else {
			hc, err = loader.Load(subject)
			if err != nil {
				return nil, err
			}

		}
	case *helmchart.Chart:
		if hc == nil {
			return nil, fmt.Errorf("passed nil helm sdk-go *Chart struct")
		}
		hc = subject
	default:
		return nil, fmt.Errorf(
			"unhandled type for inspect subject: %s (%T)",
			subject, subject,
		)

	}
	installer := action.NewInstall(&action.Configuration{})
	installer.ClientOnly = true
	installer.DryRun = true
	installer.ReleaseName = "kube-inspect"
	installer.IncludeCRDs = true
	installer.Namespace = "default"
	installer.DisableHooks = true
	release, err := installer.Run(hc, nil)
	if err != nil {
		return nil, err
	}
	resources, err := resourcesFromManifest(bytes.NewBuffer([]byte(release.Manifest)))
	if err != nil {
		return nil, err
	}
	// Unfortunately, the helm sdk-go Release.Info.Resources map is empty when
	// "installing" in dry-run mode (which is necessary to render the templates
	// but not actually install anything). So we need to manually construct the
	// set of Kubernetes resources by processing the rendered multi-document
	// YAML manifest.
	return &Chart{
		Chart:     hc,
		resources: resources,
	}, nil
}

// fetchArchive reads the tarball at the supplied URL, copies it to a temporary
// file and returns the temporary file. callers are responsible for removing
// the temporary file.
func fetchArchive(url string) (*os.File, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-ok read from %q: %d", url, resp.StatusCode)
	}

	f, err := os.CreateTemp("", filepath.Base(url))
	if err != nil {
		return nil, err
	}
	io.Copy(f, resp.Body)
	f.Seek(0, 0)
	return f, nil
}

var (
	regexDocument       = regexp.MustCompile("(?m)^---")
	regexSourceFilePath = regexp.MustCompile(`(?m)^# Source:(.*)$`)
)

// resourcesFromManifest processes the raw multi-document YAML manifest from
// the rendered Helm Chart into zero or more Kubernetes resources represented
// as unstructured.Unstructured structs.
func resourcesFromManifest(
	manifest *bytes.Buffer,
) (map[string]*unstructured.Unstructured, error) {
	sourcePaths := []string{}
	resources := map[string]*unstructured.Unstructured{}
	docs := regexDocument.Split(manifest.String(), -1)
	for _, doc := range docs {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}

		matches := regexSourceFilePath.FindStringSubmatch(doc)
		if len(matches) == 0 {
			continue
		}

		sourcePath := strings.TrimSpace(matches[1])
		sourcePaths = append(sourcePaths, sourcePath)
	}

	var err error
	var docIndex int
	decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(manifest.Bytes()), 100)
	for {
		var rawObj runtime.RawExtension
		if err = decoder.Decode(&rawObj); err != nil {
			break
		}

		obj, gvk, err := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode(rawObj.Raw, nil, nil)
		if err != nil {
			fmt.Println(manifest.String())
			return nil, err
		}
		unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			return nil, err
		}

		us := &unstructured.Unstructured{Object: unstructuredMap}
		kind := gvk.GroupKind()
		name, _, err := unstructured.NestedString(us.Object, "metadata", "name")
		if err != nil {
			return nil, err
		}
		fmt.Printf("  identified resource %s (%s)\n", name, kind)
		resources[sourcePaths[docIndex]] = us
		docIndex++
	}

	return resources, nil
}
