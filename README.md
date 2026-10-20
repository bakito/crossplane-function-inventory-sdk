# Inventory SDK for go crossplane functions

A simplified library for building Crossplane composition functions that provides type-safe resource management through
an inventory-based approach.

## Introduction

Crossplane inventory SDK eliminates the complexity of manually handling Crossplane function requests and responses by providing a
clean, tag-driven inventory system. Instead of working directly with unstructured data and managing complex resource
lifecycles, you define your resource requirements using Go struct tags and let Crossplane inventory SDK handle the rest.

Key features:

- **Type Safety**: Automatic conversion between unstructured Kubernetes resources and strongly-typed Go objects
- **Tag-Based Configuration**: Declarative resource binding through struct tags
- **Simplified API**: Clean request → inventory → response flow

## Getting Started

How your fn.go could look like

```go
package main

import (
	"context"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/response"
	"github.com/pkg/errors"
	"github.com/bakito/crossplane-function-inventory-sdk/inventory"
)

// Declare yor inventory
type Inventory struct {
	ObservedComposedNS      *corev1.Namespace           `crossplane:"observed-composed:ns"`
	ObservedComposedNSReady fnv1.Ready                  `crossplane:"observed-composed:ns"`
	ObservedComposedNSConn  inventory.ConnectionDetails `crossplane:"observed-composed:ns"`

	DesiredComposedNS      *akcv1alpha1.AzureKubernetesCluster `crossplane:"desired-composed:ns"`
	DesiredComposedNSReady fnv1.Ready                          `crossplane:"desired-composed:ns"`

	DesiredComposedMap      map[string]*corev1.ConfigMap           `crossplane:"desired-composed:cm-"`
	DesiredComposedMapReady map[string]fnv1.Ready                  `crossplane:"desired-composed:cm-"`
	DesiredComposedMapConn  map[string]inventory.ConnectionDetails `crossplane:"desired-composed:cm-"`

	ObservedComposite      *v1alpha1.Subscription      `crossplane:"observed-composite"`
	ObservedCompositeReady fnv1.Ready                  `crossplane:"observed-composite"`
	ObservedCompositeConn  inventory.ConnectionDetails `crossplane:"observed-composite"`
}

func CreateNS(inv &Inventory) error {
	inv.DesiredComposedNS = &corev1.Namespace{}
	return nil
}

// Function returns whatever response you ask it to.
type Function struct {
	fnv1.UnimplementedFunctionRunnerServiceServer
	log logging.Logger
}

// RunFunction runs the Function.
func (f *Function) RunFunction(_ context.Context, req *fnv1.RunFunctionRequest) (*fnv1.RunFunctionResponse, error) {
	rsp := response.To(req, response.DefaultTTL)

	inv := &Inventory{}
	if err := inventory.BuildFromRequest(req, f.log, inv); err != nil {
		response.Fatal(rsp, errors.Wrap(err, "failed to build inventory"))
		return rsp, nil
	}

	if err := CreateNS(inv); err != nil {
		response.Fatal(rsp, errors.Wrap(err, "failed to create namespace"))
		return rsp, nil
	}

	if err := inventory.ConvertToResponse(rsp, inv); err != nil {
		response.Fatal(rsp, errors.Wrap(err, "failed to convert inventory to response"))
		return rsp, nil
	}
	return rsp, nil
}
```

## Tag Combinations

The `crossplane` struct tag supports various combinations to handle different resource types and access patterns:

### Composite Resource

```go
// Observed Composite
ObservedComposite        *corev1.ConfigMap `crossplane:"observed-composite"`
ObservedCompositeConn    inventory.ConnectionDetails `crossplane:"observed-composite"`

// Desired Composite

// The desired composite resource can only modify the status of the resource
DesiredComposite        *corev1.ConfigMap `crossplane:"desired-composite"`
DesiredCompositeConn    inventory.ConnectionDetails `crossplane:"desired-composite"`
```

### Single Managed Resource

```go
// Access individual resources by exact name
ObservedNamespace        *corev1.Namespace                `crossplane:"observed-composed:my-namespace"`
ObservedNamespaceConn    inventory.ConnectionDetails    `crossplane:"observed-composed:my-namespace"`
// Create individual resource by exact name
DesiredNamespace        *corev1.Namespace                `crossplane:"desired-composed:my-namespace"`
DesiredNamespaceReady    fnv1.Ready                        `crossplane:"desired-composed:my-namespace"`
```

### Map-Based Managed Resource (Prefix Matching)

```go
// Access multiple resources matching a prefix
ObservedNamespace        map[string]*corev1.Namespace            `crossplane:"observed-composed:ns-"`
ObservedNamespaceConn    map[string]inventory.ConnectionDetails    `crossplane:"observed-composed:ns-"`
// Creating multiple resources matching a prefix
DesiredNamespace        map[string]*corev1.Namespace            `crossplane:"desired-composed:ns-"`
DesiredNamespaceReady    map[string]fnv1.Ready                    `crossplane:"desired-composed:ns-"`
```

### Function Input

```go
// Access function input
FunctionInput    *myapi.MyInput                `crossplane:"input"`
```

### Required Resources (Required Resources)

Requesting required resources:

```go
reqs, err := require.Requires(
require.As(RequiredResourcesServer).WithRO(&v1beta1.MSSQLServer{}).MatchName(sqlNaming.ServerName()),
require.As(RequiredResourcesElasticPool).WithRO(&v1beta1.MSSQLElasticPool{}).MatchName(sqlNaming.ElasticPoolName()),
)
if err != nil {
return rsp, err
}

// Register requirements before checking, so Crossplane knows what to fetch.
reqs.Register(rsp)

provided, err := reqs.Assure(req)
if err != nil {
response.Fatal(rsp, errors.Wrapf(err, "requirements not fulfilled"))
return rsp, nil
}
if !provided {
m.Log.Debug("Required resources not yet provided, waiting for next iteration")
return rsp, nil
}
```

Accessing required resources

```go
// Access a single required resource
RequiredPod      *corev1.Pod `crossplane:"required:pod-required"`
RequiredPodReady fnv1.Ready  `crossplane:"required:pod-required"`
// Accessing a list of required resources
RequiredSvc      []*corev1.Service `crossplane:"required:svc-required"`
RequiredSvcReady []fnv1.Ready      `crossplane:"required:svc-required"`
```

## Generating Constants

The generator scans your struct tags and creates typed constants for all resource names, helping prevent typos and
making refactoring easier.

It's recommended to setup the genearation within a `generate.go` file:

```go
//go:build generate
// +build generate

//go:generate go run -tags github.com/bakito/crossplane-function-inventory-sdk/cmd/tag-const-gen -i inventory.go
```

## Testing

The `testing` package supports two styles of table-driven tests. Both use a map keyed by the test name, with a
`Reason` that is printed above every diff.

- `ft.Case`: hand-written `Args` (request) and `Want` (response / error). Best for focused unit cases and error paths.
- `ft.GoldenFileCase`: YAML fixtures as input, and the desired output is compared against golden files. Best for
  end-to-end scenarios whose output is too large to write by hand.

### Recommended layout

```
fn.go
fn_test.go               # TestRunFunction: ft.Case, small focused cases
fn_golden_test.go        # TestRunFunctionGolden: ft.GoldenFileCase, end-to-end scenarios
testdata/
  fixtures/              # shared inputs for hand-written cases (ft.SetResourceBuilderBaseDir)
    xr.yaml
    input.yaml
    want/bucket.yaml
  golden/
    <case>/              # one self-contained directory per golden case
      xr.yaml
      input.yaml
      observed/*.yaml
      required/*.yaml
      desired/*.yaml     # desired state of a prior pipeline step
      want/              # DesiredLocation, generated with -update
```

- One directory per golden case keeps a case easy to review. `-update` only cleans up that case's own `want/`
  directory, and similar case names (e.g. `basic` and `basic-ha`) can't pick up each other's golden files.
- Hand-written cases share a small set of fixtures. Each case changes only what it needs with `WithStatus`,
  `WithStatusConditions`, `WithoutStatus` or `WithoutName`.

### Hand-written test cases

```go
package main

import (
	"context"
	"errors"
	"testing"

	"github.com/crossplane/function-sdk-go/logging"

	ft "github.com/bakito/crossplane-function-inventory-sdk/testing"
)

func TestRunFunction(t *testing.T) {
	// paths passed to ft.From(...) are relative to this directory
	ft.SetResourceBuilderBaseDir("testdata/fixtures")

	cases := map[string]ft.Case{
		"creates-bucket": {
			Reason: "A bucket should be desired when the XR asks for one.",
			Args: ft.Args{
				Ctx: context.Background(),
				Req: ft.NewRequestBuilder().
					WithInput(ft.From("input.yaml")).
					WithObservedComposite(ft.From("xr.yaml")).
					Build(),
			},
			Want: ft.Want{
				// the response builder sets a default TTL of 60s
				Rsp: ft.NewResponseBuilder().
					WithDesiredResource("bucket", ft.From("want/bucket.yaml").WithoutStatus()).
					Build(),
			},
		},
		"keeps-prior-desired": {
			Reason: "Desired state of a previous pipeline step should be kept.",
			Args: ft.Args{
				Ctx: context.Background(),
				Req: ft.NewRequestBuilder().
					WithObservedComposite(ft.From("xr.yaml")).
					WithDesiredResource("other", ft.From("other.yaml")).
					Build(),
			},
			Want: ft.Want{
				Rsp: ft.NewResponseBuilder().
					WithDesiredResource("other", ft.From("other.yaml")).
					WithDesiredResource("bucket", ft.From("want/bucket.yaml").WithoutStatus()).
					Build(),
			},
			// compare the desired composite and each desired resource as YAML
			DiffOptions: []ft.DiffOption{ft.Granular()},
		},
		"invalid-input": {
			Reason: "An invalid input should return an error.",
			Args: ft.Args{
				Ctx: context.Background(),
				Req: ft.NewRequestBuilder().
					WithInput(ft.From("input-invalid.yaml")).
					WithObservedComposite(ft.From("xr.yaml")).
					Build(),
			},
			Want: ft.Want{Err: errors.New("cannot parse input")},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			tc.Run(t, &Function{log: logging.NewNopLogger()})
		})
	}
}
```

- `Want.Rsp` is compared in full, including meta/TTL, results and conditions. With `ft.Granular()`, only the
  desired composite and the desired resources are compared.
- `Results` additionally checks the severity (and the message, if set) of each result returned.
- `Want.Err` is compared by its message. If an error is expected, the response is not checked.
- `Parallel: n` runs the same case n times at once, to detect race conditions.

### Golden file test cases

```go
package main

import (
	"errors"
	"testing"

	"github.com/crossplane/function-sdk-go/logging"

	ft "github.com/bakito/crossplane-function-inventory-sdk/testing"
)

func TestRunFunctionGolden(t *testing.T) {
	const dir = "testdata/golden"

	cases := map[string]ft.GoldenFileCase{
		"basic": {
			Reason:          "The default XR produces a bucket and a policy.",
			Composite:       dir + "/basic/xr.yaml",
			Input:           dir + "/basic/input.yaml",
			DesiredLocation: dir + "/basic/want",
		},
		"with-observed": {
			Reason:    "The observed bucket status is propagated to the XR status.",
			Composite: dir + "/with-observed/xr.yaml",
			Input:     dir + "/with-observed/input.yaml",
			Observed: map[string]string{
				"bucket": dir + "/with-observed/observed/bucket.yaml",
			},
			Required: map[string]string{
				"config": dir + "/with-observed/required/config.yaml",
			},
			// desired state produced by a prior function in the pipeline
			DesiredComposite: dir + "/with-observed/desired/xr.yaml",
			Desired: map[string]string{
				"other": dir + "/with-observed/desired/other.yaml",
			},
			DesiredLocation: dir + "/with-observed/want",
			// also verify the ready state of the desired resources
			WithReady: true,
		},
		"invalid-input": {
			Reason:          "An invalid input should return an error.",
			Composite:       dir + "/invalid-input/xr.yaml",
			Input:           dir + "/invalid-input/input.yaml",
			DesiredLocation: dir + "/invalid-input/want",
			Err:             errors.New("cannot parse input"),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			tc.Run(t, name, &Function{log: logging.NewNopLogger()})
		})
	}
}
```

The golden files are stored in `DesiredLocation`. They are named `<case-name>-<composite-name>-<suffix>.yaml`, where the
suffix is:

- the function resource key for each desired resource
- `composite` for the desired composite
- `ready` for the ready states (only with `WithReady: true`)

Characters in the case name that are not allowed in file names are replaced with `_`.

Notes:

- `DesiredLocation` must be an existing directory. Add a `.gitkeep` to an empty `want/` directory.
- `DesiredLocation` is **not** resolved against `ft.SetResourceBuilderBaseDir`, while all other paths are. Use full
  paths in golden file tests and don't set a base directory there.
- Golden files in `DesiredLocation` that match a case but are not produced by the function fail the test.

### Generate / update golden files

The golden files can be generated/updated by running the test with the `-update` flag. Golden files that are no longer
produced by the function are deleted.

`go test -v -run TestRunFunctionGolden -update`
