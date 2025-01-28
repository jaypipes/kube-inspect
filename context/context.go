// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package context

import (
	"context"
	"io"
	"os"
)

type ContextKey string

var (
	DebugKey = ContextKey("kube-inspect.debug")
	TraceKey = ContextKey("kube-inspect.trace")
)

// ContextModifier sets some value on the context
type ContextModifier func(context.Context) context.Context

// WithDebug informs kube-inspect to output extra debugging information. You
// can supply zero or more `io.Writer` objects to the function.
//
// NOTE: If no `io.Writer` objects are supplied, kube-inspect will output debug
// messages using the `fmt.Printf` function. The `fmt.Printf` function is
// *unbuffered* however unless you call `go test` with the `-v` argument, `go
// test` swallows output to stdout and does not display it unless a test fails.
//
// This means that you will only get these debug messages if you call the `go
// test` tool with the `-v` option (either as `go test -v` or with `go test
// -v=test2json`.
//
// ```go
//
//	 package example_test
//
//	 import (
//	     "testing"
//
//	     "github.com/stretchr/testify/require"
//	     kictx "github.com/jaypipes/kube-inspect/context"
//	     kihelm "github.com/jaypipes/kube-inspect/helm"
//	 )
//
//	 const (
//		    nginxChartURL = "https://charts.bitnami.com/bitnami/nginx-8.8.4.tgz"
//	 )
//
//		func TestExample(t *testing.T) {
//		    require := require.New(t)
//
//		    ctx := kictx.New(kictx.WithDebug())
//		    chart, err := kihelm.Inspect(ctx, nginxChartURL)
//		    require.Nil(err)
//		    require.Len(chart.Resources(), 3)
//		}
//
// ```
//
// If you want kube-inspect to log extra debugging information about tests and
// assertions to a different file or collecting buffer, pass it a context with
// a debug `io.Writer`:
//
// ```go
// f := ioutil.TempFile("", "mytest*.log")
// ctx := kictx.New(kictx.WithDebug(f))
// ```
//
// ```go
// var b bytes.Buffer
// w := bufio.NewWriter(&b)
// ctx := kictx.New(kictx.WithDebug(w))
// ```
//
// you can then inspect the debug "log" and do whatever you'd like with it.
func WithDebug(writers ...io.Writer) ContextModifier {
	return func(ctx context.Context) context.Context {
		if len(writers) == 0 {
			// Write to stdout when WithDebug() is called with no parameters
			writers = []io.Writer{os.Stdout}
		}
		return context.WithValue(ctx, DebugKey, writers)
	}
}

// New returns a new Context
func New(mods ...ContextModifier) context.Context {
	ctx := context.TODO()
	for _, mod := range mods {
		ctx = mod(ctx)
	}
	return ctx
}
