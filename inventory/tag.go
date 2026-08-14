package inventory

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"

	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
)

const (
	TagKey       = "crossplane"
	TagSeparator = ":"

	TagTypeInput             = "input"
	TagTypeRequired          = "required"
	TagTypeObservedComposite = "observed-composite"
	TagTypeObservedComposed  = "observed-composed"
	TagTypeDesiredComposite  = "desired-composite"
	TagTypeDesiredComposed   = "desired-composed"
)

func parseTagConfig(tag string, field reflect.StructField) (tagConfig, error) {
	parts := strings.Split(tag, TagSeparator)

	cfg := tagConfig{
		field:   field,
		tagType: parts[0],
		isSlice: field.Type.Kind() == reflect.Array || field.Type.Kind() == reflect.Slice,
		// if it is a connection details field, it is not handled as a map
		isMap:     field.Type.Kind() == reflect.Map && !isConnectionDetailsType(field.Type),
		fieldType: field.Type,
	}

	if cfg.isSlice || cfg.isMap {
		cfg.fieldType = field.Type.Elem()
	}

	cfg.isReadyField = isReadyType(cfg.fieldType)
	cfg.isConnectionsField = isConnectionDetailsType(cfg.fieldType)

	switch cfg.tagType {
	case TagTypeInput, TagTypeObservedComposite, TagTypeDesiredComposite:
		if len(parts) != 1 {
			return tagConfig{}, fmt.Errorf(
				"tag %s of field %s has incorrect format - must be: %q",
				tag, field.Name, cfg.tagType,
			)
		}
	default:
		if len(parts) != 2 {
			prefix := ""
			if cfg.isMap {
				prefix = "-prefix"
			}
			return tagConfig{}, fmt.Errorf(
				"tag %s of field %s has incorrect format - must be: \"%s%s<resource-name%s>\"",
				tag, field.Name, cfg.tagType, TagSeparator, prefix,
			)
		}
	}
	if len(parts) == 2 {
		cfg.objectKey = parts[1]
	}
	return cfg, nil
}

func validateTarget(target any) (reflect.Value, error) {
	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return reflect.Value{}, errors.New("target must be a non-nil pointer to struct")
	}
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return reflect.Value{}, errors.New("target must be a pointer to struct")
	}
	return v, nil
}

// Inject sets the values in the target struct fields based on tags and inventory data.
// It processes fields with "crossplane" tags and injects corresponding values from the inventory.
func Inject(inv Reader, targets ...any) error {
	for _, target := range targets {
		reflectValue, err := validateTarget(target)
		if err != nil {
			return err
		}

		structType := reflectValue.Type()
		for fieldIndex := range structType.NumField() {
			field := structType.Field(fieldIndex)
			if err := processFieldInjection(
				inv,
				field,
				reflectValue.Field(fieldIndex),
			); err != nil {
				return err
			}
		}
	}
	return nil
}

// processFieldInjection handles the injection of values for a single struct field.
func processFieldInjection(
	inv Reader,
	field reflect.StructField,
	fieldValue reflect.Value,
) error {
	if !fieldValue.CanSet() {
		return fmt.Errorf("cannot set field %s", field.Name)
	}

	// Handle inventory.Reader field type specially
	if field.Type == reflect.TypeFor[Reader]() {
		fieldValue.Set(reflect.ValueOf(inv))
		return nil
	}

	// Process tagged fields
	tagValue := strings.TrimSpace(field.Tag.Get(TagKey))
	if tagValue == "" {
		return nil
	}

	config, err := parseTagConfig(tagValue, field)
	if err != nil {
		return err
	}

	handler, exists := handlers[config.tagType]
	if !exists {
		return nil
	}

	newValue, ok := handler(inv, config)
	if !ok {
		return nil
	}

	return setValue(config, field, newValue, fieldValue)
}

func setValue(cfg tagConfig, field reflect.StructField, newVal, val reflect.Value) error {
	if cfg.isMap && !cfg.isReadyField && !cfg.isConnectionsField {
		newMap := reflect.MakeMap(field.Type)
		for iter := newVal.MapRange(); iter.Next(); {
			// Extract intermediate values and add type assertion check
			iterValue := iter.Value()
			iface := iterValue.Interface()
			runtimeObj, ok := iface.(runtime.Object)
			if !ok {
				return errors.New("not runtime.Object")
			}
			objValue := reflect.ValueOf(runtimeObj)

			newMap.SetMapIndex(iter.Key(), objValue)
		}
		val.Set(newMap)
	} else {
		val.Set(newVal)
	}
	return nil
}

// MappingType is a mapping of resource names to their types.
type MappingType map[string]reflect.Type

// TypeRegistry manages type validation and storage for different field categories.
type TypeRegistry struct {
	inputType     reflect.Type
	compositeType reflect.Type
	mappingTypes  MappingType
}

func (r *TypeRegistry) validateAndSetType( //nolint:gocyclo // FIXME
	cfg tagConfig,
) error {
	if isReadyType(cfg.fieldType) || isConnectionDetailsType(cfg.fieldType) {
		return nil
	}
	switch cfg.tagType {
	case TagTypeInput:
		if r.inputType != nil && r.inputType != cfg.fieldType {
			return errors.New("input type must be the same for all targets")
		}
		r.inputType = cfg.fieldType
	case TagTypeObservedComposite, TagTypeDesiredComposite:
		if r.compositeType != nil && r.compositeType != cfg.fieldType &&
			!isReadyType(cfg.fieldType) && !isConnectionDetailsType(cfg.fieldType) {
			return fmt.Errorf(
				"%s type must be the same for all targets", cfg.tagType,
			)
		}
		r.compositeType = cfg.fieldType
	case TagTypeRequired, TagTypeObservedComposed, TagTypeDesiredComposed:
		if existingType, exists := r.mappingTypes[cfg.objectKey]; exists &&
			existingType != cfg.fieldType {
			return fmt.Errorf(
				"%s types %s must be the same for all targets. Got %q and %q",
				cfg.tagType,
				cfg.objectKey,
				existingType,
				cfg.fieldType,
			)
		}
		if other := FindConflictingPrefix(r.mappingTypes, cfg.objectKey); other != "" {
			return fmt.Errorf(
				"resource name prefix must be unique for tag types [%s, %s, %s]. Got conflict with %q and %q",
				TagTypeRequired,
				TagTypeObservedComposed,
				TagTypeDesiredComposed,
				cfg.objectKey,
				other,
			)
		}
		r.mappingTypes[cfg.objectKey] = cfg.fieldType
	default:
	}
	return nil
}

func BuildInventoryOptions(targets ...any) ([]Option, error) {
	registry := &TypeRegistry{
		mappingTypes: make(MappingType),
	}

	for _, target := range targets {
		structValue, err := validateTarget(target)
		if err != nil {
			return nil, err
		}

		structType := structValue.Type()
		for field := range structType.Fields() {
			if err := processField(field, registry); err != nil {
				return nil, err
			}
		}
	}

	return buildOptions(registry), nil
}

func processField(field reflect.StructField, registry *TypeRegistry) error {
	tag := strings.TrimSpace(field.Tag.Get(TagKey))
	if tag == "" {
		return nil
	}

	cfg, err := parseTagConfig(tag, field)
	if err != nil {
		return err
	}

	if err := validateFieldTypeAssignability(field, cfg.fieldType); err != nil {
		return err
	}

	return registry.validateAndSetType(cfg)
}

func buildOptions(registry *TypeRegistry) []Option {
	var opts []Option

	if registry.inputType != nil {
		opts = append(opts, WithInput(factoryFunction(registry.inputType)))
	}

	if registry.compositeType != nil {
		opts = append(opts, WithComposite(factoryFunction(registry.compositeType)))
	}

	if len(registry.mappingTypes) > 0 {
		mapping := make(Mapping)
		for name, t := range registry.mappingTypes {
			mapping[name] = factoryFunction(t)
		}
		opts = append(opts, WithMapping(mapping))
	}

	return opts
}

func factoryFunction(fieldType reflect.Type) func() (runtime.Object, error) {
	return func() (runtime.Object, error) {
		v := reflect.New(fieldType.Elem()).Interface()
		obj, ok := v.(runtime.Object)
		if !ok {
			return nil, errors.New("type does not implement runtime.Object")
		}
		return obj, nil
	}
}

func validateFieldTypeAssignability(field reflect.StructField, fieldType reflect.Type) error {
	if !fieldType.Implements(reflect.TypeFor[runtime.Object]()) &&
		!isReadyType(fieldType) && !isConnectionDetailsType(fieldType) {
		return fmt.Errorf(
			"type %s of target struct %s must implement runtime.Object interface or be assignable to fnv1.Ready or	 inventory.ConnectionDetails",
			fieldType,
			field.Name,
		)
	}
	return nil
}

func isReadyType(fieldType reflect.Type) bool {
	return fieldType.AssignableTo(reflect.TypeFor[fnv1.Ready]())
}

func isConnectionDetailsType(fieldType reflect.Type) bool {
	return fieldType.Kind() == reflect.Map &&
		fieldType.Key().Kind() == reflect.String &&
		fieldType.AssignableTo(reflect.TypeFor[ConnectionDetails]())
}

var handlers = map[string]fieldHandler{
	TagTypeInput: func(inv Reader, cfg tagConfig) (reflect.Value, bool) {
		if ro := inv.GetInput(); ro != nil {
			return reflect.ValueOf(ro), true
		}
		return reflect.Value{}, false
	},
	TagTypeObservedComposite: func(inv Reader, cfg tagConfig) (reflect.Value, bool) {
		if ro := inv.GetObservedComposite(); ro != nil {
			return cfg.resourceValue(ro), true
		}
		return reflect.Value{}, false
	},
	TagTypeObservedComposed: func(inv Reader, cfg tagConfig) (reflect.Value, bool) {
		if cfg.isMap {
			return cfg.resourceMapValue(inv.GetObservedComposed())
		}
		if ro, ok := inv.GetObservedComposed()[cfg.objectKey]; ok {
			return cfg.resourceValue(ro), true
		}
		return reflect.Value{}, false
	},
	TagTypeDesiredComposite: func(inv Reader, cfg tagConfig) (reflect.Value, bool) {
		if ro := inv.GetDesiredComposite(); ro != nil {
			return cfg.resourceValue(ro), true
		}
		return reflect.Value{}, false
	},
	TagTypeDesiredComposed: func(inv Reader, cfg tagConfig) (reflect.Value, bool) {
		if cfg.isMap {
			return cfg.resourceMapValue(inv.GetDesiredComposed())
		}
		if ro, ok := inv.GetDesiredComposed()[cfg.objectKey]; ok {
			return cfg.resourceValue(ro), true
		}
		return reflect.Value{}, false
	},
	TagTypeRequired: func(inv Reader, cfg tagConfig) (reflect.Value, bool) {
		if req, ok := inv.GetRequirements()[cfg.objectKey]; ok {
			if cfg.isSlice {
				return cfg.resourcesValue(req), true
			}
			if len(req) == 1 {
				return cfg.resourceValue(req[0]), true
			}
		}
		return reflect.Value{}, false
	},
}

type tagConfig struct {
	field              reflect.StructField
	fieldType          reflect.Type
	tagType            string
	objectKey          string
	isSlice            bool
	isMap              bool
	isReadyField       bool
	isConnectionsField bool
}

type fieldHandler func(inv Reader, cfg tagConfig) (reflect.Value, bool)

func (cfg *tagConfig) resourcesValue(r Resources) reflect.Value {
	switch {
	case cfg.isReadyField:
		ready := make([]fnv1.Ready, len(r))
		for i, res := range r {
			ready[i] = res.Ready()
		}
		return reflect.ValueOf(ready)
	case cfg.isConnectionsField:
		cd := make([]ConnectionDetails, len(r))
		for i, res := range r {
			cd[i] = res.Connection()
		}
		return reflect.ValueOf(cd)
	default:
		slice := reflect.MakeSlice(cfg.field.Type, 0, len(r))
		for _, item := range r {
			slice = reflect.Append(slice, reflect.ValueOf(item.RuntimeObject()))
		}
		return slice
	}
}

func (cfg *tagConfig) resourceValue(r *Resource) reflect.Value {
	switch {
	case cfg.isReadyField:
		return reflect.ValueOf(r.Ready())
	case cfg.isConnectionsField:
		return reflect.ValueOf(r.Connection())
	default:
		return reflect.ValueOf(r.RuntimeObject())
	}
}

func (cfg *tagConfig) resourceMapValue(rm ResourceMap) (reflect.Value, bool) {
	if rm, ready, conn, ok := GetResourceMap[runtime.Object](rm, cfg.objectKey); ok {
		if cfg.isReadyField {
			return reflect.ValueOf(ready), true
		}
		if cfg.isConnectionsField {
			return reflect.ValueOf(conn), true
		}
		return reflect.ValueOf(rm), true
	}
	return reflect.Value{}, false
}
