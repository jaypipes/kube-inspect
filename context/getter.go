// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package context

import (
	"context"
	"io"
	"strings"
)

const (
	traceDelimiter = "/"
)

// Trace gets a context's trace name stack joined together with traceDelimiter
func Trace(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v := ctx.Value(TraceKey); v != nil {
		return strings.Join(v.([]string), traceDelimiter)
	}
	return ""
}

// TraceStack gets a context's trace name stack
func TraceStack(ctx context.Context) []string {
	if ctx == nil {
		return []string{}
	}
	if v := ctx.Value(TraceKey); v != nil {
		return v.([]string)
	}
	return []string{}
}

// Debug gets a context's Debug writer
func Debug(ctx context.Context) []io.Writer {
	if ctx == nil {
		return []io.Writer{}
	}
	if v := ctx.Value(DebugKey); v != nil {
		return v.([]io.Writer)
	}
	return []io.Writer{}
}
