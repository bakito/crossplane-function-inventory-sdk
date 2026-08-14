package testing

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"google.golang.org/protobuf/types/known/structpb"
	"gopkg.in/yaml.v3"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

var resourceBuilderBaseDir = ""

// SetResourceBuilderBaseDir defines the base directory path used for resource building operations.
func SetResourceBuilderBaseDir(dir string) {
	resourceBuilderBaseDir = dir
}

// ResourceBuilder Reads an initial resource from a YAML file and optionally adds status/conditions to it.
type ResourceBuilder struct {
	filepath    string
	status      map[string]any
	resetStatus bool
	conditions  []xpv2.Condition
	resetName   bool
}

// From Returns a new Resource Builder from a given file.
func From(file string) *ResourceBuilder {
	return NewResourceFromFile(file)
}

// NewResourceFromFile Returns a new Resource Builder.
func NewResourceFromFile(file string) *ResourceBuilder {
	path := file
	if resourceBuilderBaseDir != "" {
		path = filepath.Join(resourceBuilderBaseDir, file)
	}
	return &ResourceBuilder{
		filepath: path,
	}
}

// WithStatus sets optional status values in the resource.
func (b *ResourceBuilder) WithStatus(status map[string]any) *ResourceBuilder {
	b.status = status
	return b
}

// WithoutStatus removes the status from the object, mainly used for desired resources as we do not expect any status.
func (b *ResourceBuilder) WithoutStatus() *ResourceBuilder {
	b.status = nil
	b.resetStatus = true
	return b
}

// WithoutName removes the metadata.name from the object, mainly used for desired resources as we do not expect any name.
func (b *ResourceBuilder) WithoutName() *ResourceBuilder {
	b.resetName = true
	return b
}

// WithStatusConditions adds conditions
// Example: WithStatusConditions(xpv2.Available(), xpv2.ReconcileSuccess()).
func (b *ResourceBuilder) WithStatusConditions(conditions ...xpv2.Condition) *ResourceBuilder {
	b.conditions = conditions
	return b
}

// ToStruct returns a structpb.Struct object out of the initially provided configurations
// Function panics on error. To be used in tests only!
func (b *ResourceBuilder) ToStruct() *structpb.Struct {
	protoStruct := b.loadAndParseYAML()
	b.handleStatus(protoStruct)
	b.handleConditions(protoStruct)
	b.handleName(protoStruct)
	return protoStruct
}

func (b *ResourceBuilder) loadAndParseYAML() *structpb.Struct {
	yamlContent := b.readFile()
	rawData := b.unmarshalYAML(yamlContent)
	return b.convertToProtoStruct(rawData)
}

func (b *ResourceBuilder) readFile() []byte {
	content, err := os.ReadFile(b.filepath)
	if err != nil {
		b.handleError("error reading YAML file", err)
	}
	return content
}

func (b *ResourceBuilder) unmarshalYAML(content []byte) map[any]any {
	var rawData map[any]any
	if err := yaml.Unmarshal(content, &rawData); err != nil {
		b.handleError("error unmarshalling YAML file", err)
	}
	return rawData
}

func (b *ResourceBuilder) convertToProtoStruct(rawData map[any]any) *structpb.Struct {
	protoStruct, err := structpb.NewStruct(convertMap(rawData))
	if err != nil {
		b.handleError("error converting YAML file to structpb.Struct", err)
	}
	return protoStruct
}

func (b *ResourceBuilder) handleStatus(protoStruct *structpb.Struct) {
	if b.resetStatus {
		delete(protoStruct.GetFields(), "status")
		return
	}

	if b.status != nil {
		statusValue, err := structpb.NewValue(b.status)
		if err != nil {
			b.handleError("error converting status to structpb.Value", err)
		}
		protoStruct.Fields["status"] = statusValue
	}
}

func (b *ResourceBuilder) handleName(protoStruct *structpb.Struct) {
	if b.resetName {
		if meta, ok := protoStruct.GetFields()["metadata"]; ok {
			delete(meta.GetStructValue().GetFields(), "name")
		}
	}
}

func (b *ResourceBuilder) handleConditions(protoStruct *structpb.Struct) {
	if b.conditions == nil {
		return
	}

	conditions := &structpb.ListValue{}
	for _, c := range b.conditions {
		condition, err := conditionToValue(c)
		if err != nil {
			b.handleError("error converting conditions to structpb.Value", err)
		}
		conditions.Values = append(conditions.Values, condition)
	}

	if protoStruct.GetFields()["status"] == nil {
		protoStruct.Fields["status"], _ = structpb.NewValue(map[string]any{})
	}
	protoStruct.Fields["status"].GetStructValue().Fields["conditions"] = structpb.NewListValue(
		conditions,
	)
}

func (b *ResourceBuilder) handleError(msg string, err error) {
	panic(fmt.Errorf("%s '%s': %w", msg, b.filepath, err))
}

func conditionToValue(condition xpv2.Condition) (*structpb.Value, error) {
	jsonData, err := json.Marshal(condition)
	if err != nil {
		return nil, err
	}

	var result structpb.Value
	if err := result.UnmarshalJSON(jsonData); err != nil {
		return nil, err
	}

	return &result, nil
}

// Recursive function to convert map[any]any to map[string]any.
func convertMap(input map[any]any) map[string]any {
	output := make(map[string]any)
	for key, value := range input {
		strKey := fmt.Sprintf("%v", key) // Convert key to string
		switch v := value.(type) {
		case map[any]any:
			output[strKey] = convertMap(v) // Recursively handle nested maps
		case []any:
			output[strKey] = convertSlice(v) // Handle slices
		default:
			output[strKey] = v
		}
	}
	return output
}

// Recursive function to convert slices with map[any]any elements.
func convertSlice(input []any) []any {
	output := make([]any, len(input))
	for i, value := range input {
		switch v := value.(type) {
		case map[any]any:
			output[i] = convertMap(v) // Convert nested maps inside the slice
		case []any:
			output[i] = convertSlice(v) // Recursively handle nested slices
		default:
			output[i] = v
		}
	}
	return output
}
