[![Go Reference](https://pkg.go.dev/badge/github.com/jaypipes/kube-inspect.svg)](https://pkg.go.dev/github.com/jaypipes/kube-inspect)
[![Go Report Card](https://goreportcard.com/badge/github.com/jaypipes/kube-inspect)](https://goreportcard.com/report/github.com/jaypipes/kube-inspect)
[![Build Status](https://github.com/jaypipes/kube-inspect/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/jaypipes/kube-inspect/actions)
[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg)](CODE_OF_CONDUCT.md)

`kube-inspect` is a Go library with functionality for inspecting Kubernetes
resources and working with Helm Charts.

It wraps tedious Helm SDK and Kubernetes client-go/kubeconfig code with more
straightforward functions and structs.

# Inspect Kubernetes resources

## Inspect resources installed by a Helm Chart

Inspect Kubernetes resources installed by a Helm Chart with the
`kube-inspect/helm` package.

First, get a `kube-inspect/helm.Chart` object and then use the
`kube-inspect/helm.Chart.Resources()` method to query the Kubernetes resources
that would be installed by the Helm Chart.

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

You can filter returned Kubernetes resources by specifying one or more
`kube-inspect/kube.ResourceFilter` objects in the call to
`kube-inspect/helm.Chart.Resources()`:

```
import (
    "github.com/jaypipes/kube-inspect/kube"
)
```

```go
    for _, r := range chart.Resources(resource.WithKind("pod")) {
        fmt.Println("name:", r.GetName())
    }
```

# Inspect Helm Chart versions

Because Helm Charts may be published in an OCI registry or a "legacy" Helm
Repository, getting a list of Helm Chart versions can be frustrating. You
either need to use the Helm v3 SDK to list index files and get chart versions
or you need to use a library like ORAS to list OCI tags and attempt to get
version information from OCI artifacts.

`kube-inspect` saves you this annoying hassle.

Construct a `kube-inspect/helm.ChartLocation` using the
`kube-inspect/helm.ChartLocationFromURL()` function:

```go
import (
    "fmt"
    "log"

    helminspect "github.com/jaypipes/kube-inspect/helm"
)

func main() {
    // subject can be an OCI URL, e.g. "oci://quay.io/jetstack/charts/cert-manager",
    // or a Helm Repository "URL", e.g. "https://charts.jetstack.io/cert-manager"
    loc, err := helminspect.ChartLocationFromURL(subject)
    if err != nil {
        log.Fatalf("failed to determine Chart Location from %s: %s", subject, err)
    }
}
```

To get a list of `kube-inspect/helm.ChartVersion` objects, simply pass a
`kube-inspect/helm.ChartLocation` object to
`kube-inspect/helm.ChartVersionsFromLocation()`:

```go
    ctx := context.TODO()   
    versions, err := helminspect.ChartVersionsFromLocation(ctx, loc)
    if err != nil {
        log.Fatalf("failed to chart versions from ChartLocation %s: %s", loc, err)
    }

    for _, ver := range versions {
        fmt.Println("version:", ver.Version)
    }
}
```
