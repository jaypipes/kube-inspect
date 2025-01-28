// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package debug

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	kictx "github.com/jaypipes/kube-inspect/context"
)

type trace struct {
	name  string
	start time.Time
}

// PushTrace pushes a debug/trace name onto the debug/trace stack.
func PushTrace(
	ctx context.Context,
	name string,
) context.Context {
	writers := kictx.Debug(ctx)
	if len(writers) == 0 {
		return ctx
	}
	stack := traceStack(ctx)
	stack = append(stack, trace{name: name, start: time.Now()})
	printTraceStart(writers, stack)
	return context.WithValue(ctx, kictx.TraceKey, stack)
}

func printTraceStart(writers []io.Writer, stack []trace) {
	if len(stack) == 0 {
		return
	}
	indent := strings.Repeat(" ", len(stack))
	var tail *trace
	traceStr := ""
	if len(stack) > 0 {
		tail = &stack[len(stack)-1]
		traceStr = tail.name
	}

	msg := fmt.Sprintf(
		"%s%s%s\n",
		debugPrefix,
		indent,
		traceStr,
	)
	for _, w := range writers {
		//nolint:errcheck
		w.Write([]byte(msg))
	}
}

// PopTrace pops the last trace off the debug/trace stack.
func PopTrace(
	ctx context.Context,
) context.Context {
	writers := kictx.Debug(ctx)
	if len(writers) == 0 {
		return ctx
	}
	stack := traceStack(ctx)
	printTraceEnd(writers, stack)
	stack = stack[:len(stack)-1]
	return context.WithValue(ctx, kictx.TraceKey, stack)
}

func printTraceEnd(writers []io.Writer, stack []trace) {
	if len(stack) == 0 {
		return
	}
	indent := strings.Repeat(" ", len(stack))
	var tail *trace
	traceStr := ""
	if len(stack) > 0 {
		tail = &stack[len(stack)-1]
		traceStr = tail.name
	}

	msg := fmt.Sprintf(
		"%s%s%s (took %s)\n",
		debugPrefix,
		indent,
		traceStr,
		time.Since(tail.start),
	)
	for _, w := range writers {
		//nolint:errcheck
		w.Write([]byte(msg))
	}
}
