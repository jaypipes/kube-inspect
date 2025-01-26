[![Go Reference](https://pkg.go.dev/badge/github.com/jaypipes/kube-inspect.svg)](https://pkg.go.dev/github.com/jaypipes/kube-inspect)
[![Go Report Card](https://goreportcard.com/badge/github.com/jaypipes/kube-inspect)](https://goreportcard.com/report/github.com/jaypipes/kube-inspect)
[![Build Status](https://github.com/jaypipes/kube-inspect/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/jaypipes/kube-inspect/actions)
[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg)](CODE_OF_CONDUCT.md)

`kube-inspect` is a Go library for inspecting Kubernetes resources within raw
manifests, Helm Charts and Kustomize overlays.

## Helm Chart inspection

Inspect what is inside a Helm Chart with the `kube-inspect/helm` package.

```go
import (
    "fmt"
    "log"

    helminspect "github.com/jaypipes/kube-inspect/helm"
)

func main() {
    // subject can be a filepath, a helm go-sdk Chart object, a URL, or an
    // `io.Reader`
    chart, err := helminspect.Inspect(subject)
    if err != nil {
        log.Fatalf("failed to inspect Helm Chart %s: %s", subject, err)
    }
    for _, r := range chart.Resources() {
        fmt.Println("kind:", r.GetKind(), "name:", r.GetName())
    }
}
```
