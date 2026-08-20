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
req, err := require.Requires(
require.As(RequiredResourcesServer).WithRO(&v1beta1.MSSQLServer{}).MatchName(sqlNaming.ServerName()),
require.As(RequiredResourcesElasticPool).WithRO(&v1beta1.MSSQLElasticPool{}).MatchName(sqlNaming.ElasticPoolName()),
)
if err != nil {
return err
}

provided, err := req.Assure(req)
if !provided {
m.Log.Debug("No required resources present, exiting", "requirements", rsp.GetRequirements())
return rsp, nil
}
if err != nil {
response.Fatal(rsp, errors.Wrapf(err, "requirements not fulfilled"))
return rsp, nil
}

req.Register(rsp)


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

Golden File Testing allows verifying the resources generated by the function against golden files.

The golden files are named as follows: `<composite-name>-<function-resource-key>.yaml`.
The file for the desired composite is named `<composite-name>-composite.yaml`.

This test case will start the function and compare the desired output against all the golden files.

```go
package main

import (
	"testing"

	ft "github.com/bakito/crossplane-function-inventory-sdk/testing"

	"github.com/crossplane/function-sdk-go/logging"
)

func TestRunFunctionGolden(t *testing.T) {
	cases := map[string]ft.GoldenFileCase{
		"My Function Test": {
			// Reason for the test case
			Reason: "Test case description",
			// Composite resource input file
			Composite: "testdata/xr.yaml",
			// Function input file
			Input: "testdata/input.yaml",
			// Location of the golden files
			DesiredLocation: "testdata/golden/",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			tc.Run(t, &Function{log: logging.NewNopLogger()})
		})
	}
}
```

### Generate / update golden files

The golden files can be generated/updated by running the test with the `-update` flag.

`go test -v -update`
