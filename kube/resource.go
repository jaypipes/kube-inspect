// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package kube

import (
	"bytes"
	"context"
	"regexp"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"

	"github.com/jaypipes/kube-inspect/debug"
)

var (
	regexDocument = regexp.MustCompile("(?m)^---")
)

// ResourcesFromManifest processes the supplied buffer containing a YAML
// document with zero or more Kubernetes resource manifests and returns a slice
// of unstructured.Unstructured structs representing those Kubernetes
// resources.
func ResourcesFromManifest(
	ctx context.Context,
	manifest *bytes.Buffer,
) ([]*unstructured.Unstructured, error) {
	ctx = debug.PushTrace(ctx, "kube:resources-from-manifest")
	defer debug.PopTrace(ctx)
	res := []*unstructured.Unstructured{}
	reader := bytes.NewReader(manifest.Bytes())
	decoder := yamlutil.NewYAMLOrJSONDecoder(reader, 100)
	parser := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	for {
		var rawObj runtime.RawExtension
		if err := decoder.Decode(&rawObj); err != nil {
			break
		}
		if len(rawObj.Raw) == 0 {
			continue
		}

		obj, gvk, err := parser.Decode(rawObj.Raw, nil, nil)
		if err != nil {
			return nil, err
		}
		usMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			return nil, err
		}

		us := &unstructured.Unstructured{Object: usMap}
		kind := gvk.GroupKind()
		name, _, err := unstructured.NestedString(us.Object, "metadata", "name")
		if err != nil {
			return nil, err
		}
		debug.Printf(ctx, "identified resource %s (%s)\n", name, kind)
		res = append(res, us)
	}

	return res, nil
}
