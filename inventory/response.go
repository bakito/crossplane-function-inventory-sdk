package inventory

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"

	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/resource/composed"
	"github.com/crossplane/function-sdk-go/resource/composite"
	"github.com/crossplane/function-sdk-go/response"
)

// ConvertToResponse is a wrapper that extracts desired state and sets the response.
func ConvertToResponse(rsp *fnv1.RunFunctionResponse, sources ...any) error {
	xr, dcds, err := ExtractDesiredState(sources...)
	if err != nil {
		return err
	}

	if xr != nil {
		if err := response.SetDesiredCompositeResource(rsp, xr); err != nil {
			return err
		}
	}

	return response.SetDesiredComposedResources(rsp, dcds)
}

// ExtractDesiredState extracts composite and composed resources from sources using reflection.
func ExtractDesiredState(
	sources ...any,
) (*resource.Composite, map[resource.Name]*resource.DesiredComposed, error) {
	xr := &resource.Composite{
		Resource:          &composite.Unstructured{},
		ConnectionDetails: nil,
	}
	dcds := make(map[resource.Name]*resource.DesiredComposed)

	for _, source := range sources {
		if err := processSource(source, xr, dcds); err != nil {
			return xr, dcds, err
		}
	}

	if isEmpty(xr) {
		return nil, dcds, nil
	}

	return xr, dcds, nil
}

func isEmpty(xr *resource.Composite) bool {
	return (xr.Resource == nil || xr.Resource.Object == nil) && len(xr.ConnectionDetails) == 0 &&
		xr.Ready == ""
}

// processSource processes a single source struct using reflection.
func processSource(
	source any,
	xr *resource.Composite,
	dcds map[resource.Name]*resource.DesiredComposed,
) error {
	structType := reflect.TypeOf(source)
	structValue := reflect.ValueOf(source)

	// Handle pointer types
	if structType.Kind() == reflect.Pointer {
		structType = structType.Elem()
		structValue = structValue.Elem()
	}

	if structType.Kind() != reflect.Struct {
		return fmt.Errorf("expected struct, got %v", structType.Kind())
	}

	// Process each field in the struct
	for i := range structType.NumField() {
		field := structType.Field(i)
		fieldValue := structValue.Field(i)

		crossplaneTag := field.Tag.Get(TagKey)
		if crossplaneTag == "" {
			continue
		}

		if err := processFieldResponse(fieldValue, crossplaneTag, xr, dcds); err != nil {
			return err
		}
	}

	return nil
}

// processFieldResponse processes a single field based on its crossplane tag.
func processFieldResponse(
	fieldValue reflect.Value,
	crossplaneTag string,
	xr *resource.Composite,
	dcds map[resource.Name]*resource.DesiredComposed,
) error {
	switch {
	case crossplaneTag == TagTypeDesiredComposite:
		return processCompositeField(fieldValue, xr)
	case strings.HasPrefix(crossplaneTag, TagTypeDesiredComposed):
		return processComposedField(fieldValue, crossplaneTag, dcds)
	default:
		return nil
	}
}

// processCompositeField handles desired composite resource fields.
func processCompositeField(fieldValue reflect.Value, xr *resource.Composite) error {
	o, ok := fieldValue.Interface().(runtime.Object)
	if !ok {
		return nil
	}
	if isNilOrEmpty(o) {
		// We return early here as parsing a nil object doesn't make sense so we just skip it
		return nil
	}

	un, err := runtime.DefaultUnstructuredConverter.ToUnstructured(o)
	if err != nil {
		return err
	}

	// A function can’t change: The metadata or spec of the composite resource.
	// https://docs.crossplane.io/latest/composition/compositions/#desired-state
	// Only status can be changed
	delete(un, "metadata")
	delete(un, "spec")
	xr.Resource.Object = un

	return nil
}

// processComposedField handles desired composed resource fields (both maps and single resources).
func processComposedField(
	fieldValue reflect.Value,
	crossplaneTag string,
	dcds map[resource.Name]*resource.DesiredComposed,
) error {
	if fieldValue.Kind() == reflect.Map {
		return processComposedMap(fieldValue, dcds)
	}

	return processComposedSingle(fieldValue, crossplaneTag, dcds)
}

// processComposedMap handles map-based composed resources.
func processComposedMap(
	fieldValue reflect.Value,
	dcds map[resource.Name]*resource.DesiredComposed,
) error {
	keys := fieldValue.MapKeys()
	slices.SortFunc(keys, func(a, b reflect.Value) int {
		return strings.Compare(a.String(), b.String())
	})

	for _, key := range keys {
		k, ok := key.Interface().(string)
		if !ok {
			return errors.New("map key must be string")
		}
		value := fieldValue.MapIndex(key).Interface()

		if err := processComposedResource(k, value, dcds); err != nil {
			return err
		}
	}
	return nil
}

// processComposedSingle handles single composed resources.
func processComposedSingle(
	fieldValue reflect.Value,
	crossplaneTag string,
	dcds map[resource.Name]*resource.DesiredComposed,
) error {
	// Extract resource name from tag
	parts := strings.Split(crossplaneTag, ":")
	if len(parts) < 2 {
		return fmt.Errorf("invalid desired-composed tag format: %s", crossplaneTag)
	}

	resourceName := parts[1]
	return processComposedResource(resourceName, fieldValue.Interface(), dcds)
}

// processComposedResource handles a single composed resource (either from map or single field).
func processComposedResource(
	resourceName string,
	value any,
	dcds map[resource.Name]*resource.DesiredComposed,
) error {
	if isNilOrEmpty(value) {
		// We return early here as parsing a nil object doesn't make sense so we just skip it
		return nil
	}
	switch v := value.(type) {
	case runtime.Object:
		return addComposedObject(resourceName, v, dcds)
	case fnv1.Ready:
		return addComposedReady(resourceName, v, dcds)
	default:
		return nil
	}
}

// addComposedObject adds or updates a composed resource object.
func addComposedObject(
	resourceName string,
	obj runtime.Object,
	dcds map[resource.Name]*resource.DesiredComposed,
) error {
	co, err := composed.From(obj)
	if err != nil {
		return err
	}

	// Delete status as that should never be written
	delete(co.Object, "status")

	rName := resource.Name(resourceName)
	dc, exists := dcds[rName]
	if exists {
		dc.Resource = co
	} else {
		cd := resource.NewDesiredComposed()
		cd.Resource = co
		dcds[rName] = cd
	}

	return nil
}

// addComposedReady adds or updates a composed resource ready state.
func addComposedReady(
	resourceName string,
	ready fnv1.Ready,
	dcds map[resource.Name]*resource.DesiredComposed,
) error {
	rName := resource.Name(resourceName)
	dc, exists := dcds[rName]
	var rspReady resource.Ready
	switch ready {
	case fnv1.Ready_READY_TRUE:
		rspReady = resource.ReadyTrue
	case fnv1.Ready_READY_FALSE:
		rspReady = resource.ReadyFalse
	default:
		rspReady = resource.ReadyUnspecified
	}
	if exists {
		dc.Ready = rspReady
	} else {
		cd := resource.NewDesiredComposed()
		cd.Ready = rspReady
		dcds[rName] = cd
	}

	return nil
}

// isNil checks if an any/interface{} value is truly nil
// This handles the gotcha where typed nil values aren't equal to nil.
func isNil(v any) bool {
	if v == nil {
		return true
	}

	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
		reflect.Pointer, reflect.Slice, reflect.UnsafePointer:
		return rv.IsNil()
	default:
		return false
	}
}

// isNilOrEmpty checks if a value is nil or represents an empty/zero value.
func isNilOrEmpty(v any) bool {
	if isNil(v) {
		return true
	}

	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.String:
		return rv.String() == ""
	case reflect.Slice, reflect.Map, reflect.Array:
		return rv.Len() == 0
	default:
		return rv.IsZero()
	}
}
