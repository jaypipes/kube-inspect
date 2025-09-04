package kube

import (
	"bytes"
	"fmt"
	"io"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
)

// ToUnstructured takes a variadic subject that can be a `[]byte` or
// `io.Reader` representing a YAML document/manifest or a `runtime.Object` and
// converts it to an `*unstructured.Unstructured`.
//
// If the supplied subject is a `[]byte` or `io.Reader`, only a single YAML
// document is read from the input stream.
func ToUnstructured(
	subject any,
) (*unstructured.Unstructured, error) {
	switch subject := subject.(type) {
	case *unstructured.Unstructured:
		return subject, nil
	case runtime.Object:
		us, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(subject)
		return &unstructured.Unstructured{Object: us}, nil
	case []byte:
		return unstructuredFromBytes(subject)
	case io.Reader:
		b, err := io.ReadAll(subject)
		if err != nil {
			return nil, err
		}
		return unstructuredFromBytes(b)
	}
	return nil, nil
}

func unstructuredFromBytes(subject []byte) (*unstructured.Unstructured, error) {
	reader := bytes.NewReader(subject)
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

		obj, _, err := parser.Decode(rawObj.Raw, nil, nil)
		if err != nil {
			return nil, err
		}
		usMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			return nil, err
		}

		return &unstructured.Unstructured{Object: usMap}, nil
	}
	return nil, nil
}

// ToSliceUnstructured returns a `[]*unstructured.Unstructured` given a subject
// that is a `[]runtime.Object`, `*unstructured.UnstructuredList`, or
// `[]*unstructured.Unstructured`
func ToSliceUnstructured(
	subject any,
) ([]*unstructured.Unstructured, error) {
	switch subject := subject.(type) {
	case []*unstructured.Unstructured:
		return subject, nil
	case []runtime.Object:
		res := make([]*unstructured.Unstructured, len(subject))
		for x, item := range subject {
			// *unstructured.Unstructured implements runtime.Object...
			res[x] = item.(*unstructured.Unstructured)
		}
		return res, nil
	case *unstructured.UnstructuredList:
		res := make([]*unstructured.Unstructured, len(subject.Items))
		for x, item := range subject.Items {
			res[x] = &item
		}
		return res, nil
	default:
		return nil, fmt.Errorf(
			"unable to convert %T to []*unstructured.Unstructured", subject,
		)
	}
}
