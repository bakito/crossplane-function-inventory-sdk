package inventory

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"

	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
)

type (
	MappingFunc = func() (runtime.Object, error)
	Mapping     map[string]MappingFunc

	ResourceMap  = map[string]*Resource
	Resources    []*Resource
	ResourcesMap map[string]Resources

	ConnectionDetails = map[string][]byte
)

func (m Mapping) validate() error {
	for k := range m {
		if other := FindConflictingPrefix(m, k); other != "" {
			return fmt.Errorf(
				"mapping name prefix must be unique. Got conflict with %q and %q", k, other)
		}
	}
	return nil
}

type Resource struct {
	ro         runtime.Object
	ready      fnv1.Ready
	connection ConnectionDetails
}

func (r *Resource) Ready() fnv1.Ready {
	return r.ready
}

func (r *Resource) RuntimeObject() runtime.Object {
	return r.ro
}

func (r *Resource) Connection() ConnectionDetails {
	return r.connection
}

type inventory struct {
	request *fnv1.RunFunctionRequest

	// Function Input
	input runtime.Object

	// XRD from Cluster
	observedComposite *Resource
	observedComposed  ResourceMap

	// XRD modified from previous functions
	desiredComposite *Resource
	desiredComposed  ResourceMap

	// Required Resources
	requirements ResourcesMap
}

// Add these methods to the inventory struct:

func (i *inventory) GetDesiredComposite() *Resource {
	return i.desiredComposite
}

func (i *inventory) GetObservedComposite() *Resource {
	return i.observedComposite
}

func (i *inventory) GetDesiredComposed() ResourceMap {
	return i.desiredComposed
}

func (i *inventory) GetObservedComposed() ResourceMap {
	return i.observedComposed
}

func (i *inventory) GetRequirements() ResourcesMap {
	return i.requirements
}

func (i *inventory) GetInput() runtime.Object {
	return i.input
}
