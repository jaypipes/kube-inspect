// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package helm

import (
	"context"
	"strings"

	"github.com/jaypipes/kube-inspect/debug"
	"github.com/santhosh-tekuri/jsonschema"
)

const (
	latestJSONSchemaDraftURL = "https://json-schema.org/schema"
)

// loadValuesSchema examines the Helm Chart's values JSONSChema and loads a
// jsonschema.Schema object describing that document.
//
// We specifically do NOT use the same JSONSchema parsing/validation library
// that the Helm sdk uses (https://github.com/xeipuuv/gojsonschema) because
// that library has no introspection ability for the schema *itself*. We need
// to be able to search through the JSONSchema's properties and do type/name
// inspection.  We don't need to validate the JSONSchema (Helm already does
// that).
func (c *Chart) loadValuesSchema(
	ctx context.Context,
) (*jsonschema.Schema, error) {
	hc := c.Chart
	if hc == nil || hc.Schema == nil {
		return nil, nil
	}
	ctx = debug.PushTrace(ctx, "helm:chart:load-jsonschema")
	defer debug.PopTrace(ctx)
	comp := jsonschema.NewCompiler()
	comp.ExtractAnnotations = true
	err := comp.AddResource(
		latestJSONSchemaDraftURL,
		strings.NewReader(string(hc.Schema)),
	)
	if err != nil {
		return nil, err
	}
	return comp.Compile(latestJSONSchemaDraftURL)
}
