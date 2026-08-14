package inventory

import (
	"k8s.io/apimachinery/pkg/runtime"
)

// Ensure inventory implements Reader interfaces.
var _ Reader = &inventory{}

// Reader provides read access to inventory data.
type Reader interface {
	// GetDesiredComposite returns the desired composite resource
	GetDesiredComposite() *Resource

	// GetObservedComposite returns the observed composite resource
	GetObservedComposite() *Resource

	// GetDesiredComposed returns the map of desired composed resources
	GetDesiredComposed() ResourceMap

	// GetObservedComposed returns the map of observed composed resources
	GetObservedComposed() ResourceMap

	// GetRequirements returns the map of extra resources
	GetRequirements() ResourcesMap
	// GetInput returns the input
	GetInput() runtime.Object
}
