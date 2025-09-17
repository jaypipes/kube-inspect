// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package diff

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/gonvenience/bunt"
	"github.com/gonvenience/ytbx"
	"github.com/homeport/dyff/pkg/dyff"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type DiffOptions struct{}

type DiffOption func(*DiffOptions)

// Diff describes differences between two things.
//
// NOTE(jaypipes): wrapping the dyff.Report because I'm not sold on the ytbx
// and dyff libraries long-term...
type Diff struct {
	dyff.Report
}

// ResourcesDiff describes the differences between two collections of
// Kubernetes Resources.
type ResourcesDiff struct {
	// Changed is a map, keyed by full resource name
	// (APIGroupVersion/ResourceName) of resources different between Charts A
	// and B.
	Changed map[string]Diff
	// Added contains resources present in Chart B that are not present in
	// Chart A.
	Added []*unstructured.Unstructured
	// Removed contains resources present in Chart A that are not present in
	// Chart B.
	Removed []*unstructured.Unstructured
	// Unchanged contains resources that are exactly the same in both Charts.
	Unchanged []*unstructured.Unstructured
}

// String returns a formatted string containing the diff of the compared
// documents.
func (d *Diff) String() string {
	bunt.SetColorSettings(bunt.OFF, bunt.OFF)
	reportWriter := &dyff.DiffSyntaxReport{
		PathPrefix:            "@@",
		RootDescriptionPrefix: "#",
		ChangeTypePrefix:      "!",
		HumanReport: dyff.HumanReport{
			Report:                d.Report,
			Indent:                0,
			DoNotInspectCerts:     false,
			NoTableStyle:          true,
			OmitHeader:            true,
			UseGoPatchPaths:       false,
			MinorChangeThreshold:  0.1,
			MultilineContextLines: 2,
			PrefixMultiline:       true,
		},
	}

	var b bytes.Buffer

	_ = reportWriter.WriteReport(&b)
	return strings.TrimSpace(b.String())
}

type Diffable interface {
	[]byte | *unstructured.Unstructured
}

// New returns a `*Diff` that compares two supplied things.
func New[T Diffable](
	a, b T,
	opt ...DiffOption,
) (*Diff, error) {
	var af, bf ytbx.InputFile
	switch a := any(a).(type) {
	case []byte:
		adoc, err := ytbx.LoadYAMLDocuments(a)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to load YAML document: %w",
				err,
			)
		}
		af = ytbx.InputFile{Documents: adoc}
	case *unstructured.Unstructured:
		abytes, err := yaml.Marshal(a)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to marshal resource A: %w",
				err,
			)
		}
		adoc, err := ytbx.LoadYAMLDocuments(abytes)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to load YAML document: %w",
				err,
			)
		}
		af = ytbx.InputFile{Documents: adoc}
	}
	switch b := any(b).(type) {
	case []byte:
		bdoc, err := ytbx.LoadYAMLDocuments(b)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to load YAML document: %w",
				err,
			)
		}
		bf = ytbx.InputFile{Documents: bdoc}
	case *unstructured.Unstructured:
		bbytes, err := yaml.Marshal(b)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to marshal resource A: %w",
				err,
			)
		}
		bdoc, err := ytbx.LoadYAMLDocuments(bbytes)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to load YAML document: %w",
				err,
			)
		}
		bf = ytbx.InputFile{Documents: bdoc}
	}
	res, err := dyff.CompareInputFiles(
		af, bf,
		dyff.IgnoreOrderChanges(false),
		dyff.IgnoreWhitespaceChanges(false),
		dyff.KubernetesEntityDetection(true),
		dyff.DetectRenames(true),
	)
	if err != nil {
		return nil, err
	}
	return &Diff{
		res,
	}, nil
}
