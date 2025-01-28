// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package debug

import (
	"context"
	"io"

	kictx "github.com/jaypipes/kube-inspect/context"
)

// traceStack gets a context's trace stack
func traceStack(ctx context.Context) []trace {
	if ctx == nil {
		return []trace{}
	}
	if v := ctx.Value(kictx.TraceKey); v != nil {
		return v.([]trace)
	}
	return []trace{}
}

// debug gets a context's Debug writer
func debugWriters(ctx context.Context) []io.Writer {
	if ctx == nil {
		return []io.Writer{}
	}
	if v := ctx.Value(kictx.DebugKey); v != nil {
		return v.([]io.Writer)
	}
	return []io.Writer{}
}
