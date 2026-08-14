package testing

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
)

// RunFunctionRequestBuilder helps build RunFunctionRequest objects.
type RunFunctionRequestBuilder struct {
	request *fnv1.RunFunctionRequest
}

// NewRequestBuilder creates a new builder instance.
func NewRequestBuilder() *RunFunctionRequestBuilder {
	return &RunFunctionRequestBuilder{
		request: &fnv1.RunFunctionRequest{
			Observed: &fnv1.State{
				Resources: make(map[string]*fnv1.Resource),
			},
		},
	}
}

// WithMetaTag adds a meta tag.
func (b *RunFunctionRequestBuilder) WithMetaTag(tag string) *RunFunctionRequestBuilder {
	if b.request.GetMeta() == nil {
		b.request.Meta = &fnv1.RequestMeta{}
	}
	if tag != "" {
		b.request.Meta.Tag = tag
	}
	return b
}

// Woc adds an observed composite resource.
func (b *RunFunctionRequestBuilder) Woc(
	composite *ResourceBuilder,
) *RunFunctionRequestBuilder {
	return b.WithObservedComposite(composite)
}

// WithObservedComposite adds an observed composite resource.
func (b *RunFunctionRequestBuilder) WithObservedComposite(
	composite *ResourceBuilder,
) *RunFunctionRequestBuilder {
	if b.request.GetObserved() == nil {
		b.request.Observed = &fnv1.State{}
	}
	b.request.Observed.Composite = &fnv1.Resource{
		Resource: composite.ToStruct(),
	}
	return b
}

// Wor adds an observed resource with the given name.
func (b *RunFunctionRequestBuilder) Wor(
	name string,
	resource *ResourceBuilder,
	ready ...fnv1.Ready, // optional - first ready is used if provided
) *RunFunctionRequestBuilder {
	return b.WithObservedResource(name, resource, ready...)
}

// WithObservedResource adds an observed resource with the given name.
func (b *RunFunctionRequestBuilder) WithObservedResource(
	name string,
	resource *ResourceBuilder,
	ready ...fnv1.Ready, // optional - first ready is used if provided
) *RunFunctionRequestBuilder {
	if _, ok := b.request.GetObserved().GetResources()[name]; ok {
		panic(fmt.Sprintf("ObservedResource %q already exists", name))
	}
	if b.request.GetObserved() == nil {
		b.request.Observed = &fnv1.State{}
	}
	if b.request.Observed.Resources == nil {
		b.request.Observed.Resources = make(map[string]*fnv1.Resource)
	}
	b.request.Observed.Resources[name] = &fnv1.Resource{Resource: resource.ToStruct()}
	if len(ready) > 0 {
		b.request.Observed.Resources[name].Ready = ready[0]
	}
	return b
}

// WithRequiredResources adds required resources with the given name.
func (b *RunFunctionRequestBuilder) WithRequiredResources(
	name string,
	resources ...*structpb.Struct,
) *RunFunctionRequestBuilder {
	if b.request.GetRequiredResources() == nil {
		b.request.RequiredResources = make(map[string]*fnv1.Resources)
	}
	if _, ok := b.request.GetRequiredResources()[name]; ok {
		panic(fmt.Sprintf("RequiredResource %q already exists", name))
	}
	b.request.RequiredResources[name] = &fnv1.Resources{
		Items: make([]*fnv1.Resource, len(resources)),
	}
	for i, resource := range resources {
		b.request.RequiredResources[name].Items[i] = &fnv1.Resource{Resource: resource}
	}
	return b
}

// WithInput sets the input resource.
func (b *RunFunctionRequestBuilder) WithInput(input *ResourceBuilder) *RunFunctionRequestBuilder {
	b.request.Input = input.ToStruct()
	return b
}

// Build returns the constructed RunFunctionRequest.
func (b *RunFunctionRequestBuilder) Build() *fnv1.RunFunctionRequest {
	return b.request
}
