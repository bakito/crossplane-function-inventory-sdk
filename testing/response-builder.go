package testing

import (
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"

	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
)

// RunFunctionResponseBuilder helps build RunFunctionResponse objects.
type RunFunctionResponseBuilder struct {
	response *fnv1.RunFunctionResponse
}

// NewResponseBuilder creates a new builder instance.
func NewResponseBuilder() *RunFunctionResponseBuilder {
	return &RunFunctionResponseBuilder{
		response: &fnv1.RunFunctionResponse{
			Meta: &fnv1.ResponseMeta{
				Ttl: durationpb.New(60 * time.Second),
			},
		},
	}
}

// WithMetaTag adds a meta tag.
func (b *RunFunctionResponseBuilder) WithMetaTag(tag string) *RunFunctionResponseBuilder {
	if b.response.GetMeta() == nil {
		b.response.Meta = &fnv1.ResponseMeta{}
	}
	if tag != "" {
		b.response.Meta.Tag = tag
	}
	return b
}

// WithTTL sets the TTL in the response meta.
func (b *RunFunctionResponseBuilder) WithTTL(duration time.Duration) *RunFunctionResponseBuilder {
	if b.response.GetMeta() == nil {
		b.response.Meta = &fnv1.ResponseMeta{}
	}
	b.response.Meta.Ttl = durationpb.New(duration)
	return b
}

// WithDesiredComposite sets the desired composite resource.
func (b *RunFunctionResponseBuilder) WithDesiredComposite(
	composite *structpb.Struct,
) *RunFunctionResponseBuilder {
	if b.response.GetDesired() == nil {
		b.response.Desired = &fnv1.State{}
	}
	b.response.Desired.Composite = &fnv1.Resource{
		Resource: composite,
	}

	return b
}

// Wdr adds a desired resource with the given name.
func (b *RunFunctionResponseBuilder) Wdr(
	name string,
	res *ResourceBuilder,
	ready ...fnv1.Ready, // optional - first ready is used if provided
) *RunFunctionResponseBuilder {
	return b.WithDesiredResource(name, res, ready...)
}

// WithDesiredResource adds a desired resource with the given name.
func (b *RunFunctionResponseBuilder) WithDesiredResource(
	name string,
	res *ResourceBuilder,
	ready ...fnv1.Ready, // optional - first ready is used if provided
) *RunFunctionResponseBuilder {
	if b.response.GetDesired() == nil {
		b.response.Desired = &fnv1.State{}
	}
	if b.response.Desired.Resources == nil {
		b.response.Desired.Resources = make(map[string]*fnv1.Resource)
	}
	b.response.Desired.Resources[name] = &fnv1.Resource{
		Resource: res.ToStruct(),
	}
	if len(ready) > 0 {
		b.response.Desired.Resources[name].Ready = ready[0]
	}
	return b
}

// WithResult adds a result to the response.
func (b *RunFunctionResponseBuilder) WithResult(
	severity fnv1.Severity,
	message string,
	target fnv1.Target,
) *RunFunctionResponseBuilder {
	result := &fnv1.Result{
		Severity: severity,
		Message:  message,
	}
	if target.Enum() != nil {
		result.Target = target.Enum()
	}
	b.response.Results = append(b.response.Results, result)
	return b
}

// WithCondition adds a condition to the response.
func (b *RunFunctionResponseBuilder) WithCondition(
	type_ string,
	status fnv1.Status,
	reason string,
	target fnv1.Target,
) *RunFunctionResponseBuilder {
	condition := &fnv1.Condition{
		Type:   type_,
		Status: status,
		Reason: reason,
	}
	if target.Enum() != nil {
		condition.Target = target.Enum()
	}
	b.response.Conditions = append(b.response.Conditions, condition)
	return b
}

// WithNormalResult adds a normal severity result.
func (b *RunFunctionResponseBuilder) WithNormalResult(message string) *RunFunctionResponseBuilder {
	return b.WithResult(
		fnv1.Severity_SEVERITY_NORMAL,
		message,
		fnv1.Target_TARGET_COMPOSITE_AND_CLAIM,
	)
}

// WithRequiredResources adds extra resources with the given name.
func (b *RunFunctionResponseBuilder) WithRequiredResources(
	name string,
	selector *fnv1.ResourceSelector,
) *RunFunctionResponseBuilder {
	if b.response.GetRequirements() == nil {
		b.response.Requirements = &fnv1.Requirements{}
	}
	if b.response.GetRequirements().GetResources() == nil {
		b.response.Requirements.Resources = make(map[string]*fnv1.ResourceSelector)
	}
	if _, ok := b.response.GetRequirements().GetResources()[name]; ok {
		panic(fmt.Sprintf("Requirement %q already exists", name))
	}
	b.response.Requirements.Resources[name] = selector
	return b
}

// Build returns the constructed RunFunctionResponse.
func (b *RunFunctionResponseBuilder) Build() *fnv1.RunFunctionResponse {
	return b.response
}
