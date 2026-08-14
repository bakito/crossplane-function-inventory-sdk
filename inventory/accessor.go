package inventory

import (
	"strings"

	"k8s.io/apimachinery/pkg/runtime"

	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
)

// HasResource checks whether the inventory has resource of the given key.
func HasResource(inventory ResourceMap, resourceKey string) bool {
	_, exists := inventory[resourceKey]
	return exists
}

// GetResource retrieves a resource with a given key of type T from the inventory map.
func GetResource[T runtime.Object](
	inventory ResourceMap,
	resourceKey string,
) (T, fnv1.Ready, map[string][]byte, bool) {
	var result T
	res, exists := inventory[resourceKey]
	if !exists {
		return result, fnv1.Ready_READY_UNSPECIFIED, nil, false
	}

	// Safe type assertion
	typedResource, ok := GetTyped[T](res.RuntimeObject())
	if !ok {
		return result, fnv1.Ready_READY_UNSPECIFIED, nil, false
	}

	return typedResource, res.Ready(), res.Connection(), true
}

// GetResourceMap retrieves a resource of type T from the inventory map.
func GetResourceMap[T runtime.Object](
	inventory ResourceMap,
	objectKey string,
) (map[string]T, map[string]fnv1.Ready, map[string]map[string][]byte, bool) {
	rm := make(map[string]T)
	ready := make(map[string]fnv1.Ready)
	conn := make(map[string]map[string][]byte)

	for key, res := range inventory {
		if strings.HasPrefix(key, objectKey) {
			// Safe type assertion
			typedResource, ok := GetTyped[T](res.RuntimeObject())
			if ok {
				rm[key] = typedResource
				ready[key] = res.Ready()
				conn[key] = res.Connection()
			}
		}
	}

	return rm, ready, conn, true
}

// CountResources returns the number of resources stored in the inventory for the given key.
// If the key is not found in the inventory, it returns 0.
func CountResources(inventory ResourcesMap, resourceKey string) int {
	resources, exists := inventory[resourceKey]
	if !exists {
		return 0
	}
	return len(resources)
}

// GetResources retrieves resources of type []T from the inventory map.
func GetResources[T runtime.Object](
	inventory ResourcesMap,
	resourceKey string,
) ([]T, []fnv1.Ready, []map[string]byte, bool) {
	res, exists := inventory[resourceKey]
	if !exists {
		return nil, nil, nil, false
	}
	result := make([]T, len(res))
	ready := make([]fnv1.Ready, len(res))
	conn := make([]map[string]byte, len(res))

	for i, r := range res {
		// Safe type assertion
		typedResource, ok := GetTyped[T](r.RuntimeObject())
		if !ok {
			return nil, nil, nil, false
		}
		result[i] = typedResource
		ready[i] = r.Ready()
	}

	return result, ready, conn, true
}

func GetTyped[T runtime.Object](c runtime.Object) (T, bool) {
	typedResource, ok := c.(T)
	return typedResource, ok
}
