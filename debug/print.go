// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package debug

import (
	"context"
	"fmt"
	"strings"
)

const (
	debugPrefix    = "[kube-inspect]"
	traceDelimiter = "/"
)

// Printf writes a message with optional message arguments to the context's
// Debug output.
func Printf(
	ctx context.Context,
	format string,
	args ...interface{},
) {
	writers := debugWriters(ctx)
	if len(writers) == 0 {
		return
	}

	msg := fmt.Sprintf(normalizeFormat(ctx, format), args...)
	for _, w := range writers {
		//nolint:errcheck
		w.Write([]byte(msg))
	}
}

// Println writes a message with optional message arguments to the context's
// Debug output, ensuring there is a newline in the message line.
func Println(
	ctx context.Context,
	format string,
	args ...interface{},
) {
	writers := debugWriters(ctx)
	if len(writers) == 0 {
		return
	}

	format = strings.TrimSuffix(
		normalizeFormat(ctx, format),
		"\n",
	) + "\n"
	msg := fmt.Sprintf(format, args...)
	for _, w := range writers {
		//nolint:errcheck
		w.Write([]byte(msg))
	}
}

func normalizeFormat(ctx context.Context, format string) string {
	stack := traceStack(ctx)
	indent := strings.Repeat(" ", len(stack))

	return fmt.Sprintf(
		"%s%s>> %s",
		debugPrefix,
		indent,
		strings.TrimPrefix(
			strings.TrimPrefix(
				format, debugPrefix,
			),
			" ",
		),
	)
}
