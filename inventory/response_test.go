package inventory_test

import (
	gt "testing"

	"github.com/bakito/crossplane-function-inventory-sdk/inventory"
	"github.com/bakito/crossplane-function-inventory-sdk/testing"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/resource/composed"
	"github.com/crossplane/function-sdk-go/resource/composite"
)

// CompositeBuilder provides a fluent interface for building resource.Composite objects.
type CompositeBuilder struct {
	composite resource.Composite
	t         *gt.T
}

// NewComposite creates a new CompositeBuilder with default values.
func NewComposite(t *gt.T) *CompositeBuilder {
	t.Helper()
	return &CompositeBuilder{
		t: t,
		composite: resource.Composite{
			Resource:          &composite.Unstructured{},
			ConnectionDetails: nil,
		},
	}
}

// WithObject sets a complete runtime object as resource.
func (cb *CompositeBuilder) WithObject(obj runtime.Object) *CompositeBuilder {
	un, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	require.NoError(cb.t, err)

	// A function can’t change: The metadata or spec of the composite resource.
	// https://docs.crossplane.io/latest/composition/compositions/#desired-state
	// Only status can be changed
	delete(un, "metadata")
	delete(un, "spec")
	cb.composite.Resource.Object = un
	return cb
}

// WithConnectionDetail adds a single connection detail.
func (cb *CompositeBuilder) WithConnectionDetail(key string, value []byte) *CompositeBuilder {
	if cb.composite.ConnectionDetails == nil {
		cb.composite.ConnectionDetails = make(resource.ConnectionDetails)
	}
	cb.composite.ConnectionDetails[key] = value
	return cb
}

func (cb *CompositeBuilder) Build() *resource.Composite {
	return &cb.composite
}

// fakeRuntimeObject implements runtime.Object but deliberately does not
// implement client.Object (no embedded metav1.ObjectMeta, so no
// GetName/SetLabels/etc.). It proves ExtractDesiredState no longer silently
// drops runtime.Object-only types the way it did when response.go's type
// assertions required the stricter client.Object.
type fakeRuntimeObject struct {
	//nolint:revive // this is valid
	metav1.TypeMeta `json:",inline"`
	Foo             string `json:"foo"`
}

func (f *fakeRuntimeObject) DeepCopyObject() runtime.Object {
	cp := *f
	return &cp
}

func TestConvertToResponse(t *gt.T) {
	require.NoError(t, corev1.SchemeBuilder.AddToScheme(composed.Scheme))

	tests := []struct {
		name    string
		targets []any
		xr      *resource.Composite
		dsdc    map[resource.Name]*resource.DesiredComposed
		err     error
	}{
		{
			name: "Empty Inventory",
			targets: []any{
				struct {
					DesiredComposite     *corev1.ConfigMap                      `crossplane:"desired-composite"`
					DesiredCompositeConn map[string]inventory.ConnectionDetails `crossplane:"desired-composite"`

					DesiredComposedMap      map[string]*corev1.ConfigMap `crossplane:"desired-composed:cm-"`
					DesiredComposedMapReady map[string]resource.Ready    `crossplane:"desired-composed:cm-"`
					DesiredComposed         *corev1.ConfigMap            `crossplane:"desired-composed:cm-test"`
					DesiredComposedReady    resource.Ready               `crossplane:"desired-composed:cm-test"`
				}{},
			},
			dsdc: map[resource.Name]*resource.DesiredComposed{},
			xr:   nil,
		},
		{
			name: "DesiredComposite resource mapping",
			targets: []any{
				struct {
					DesiredComposite *corev1.ConfigMap `crossplane:"desired-composite"`
				}{
					DesiredComposite: &corev1.ConfigMap{
						Immutable: new(true),
					},
				},
			},
			dsdc: map[resource.Name]*resource.DesiredComposed{},
			xr: NewComposite(t).WithObject(&corev1.ConfigMap{
				Immutable: new(true),
			}).Build(),
		},
		{
			name: "DesiredComposite resource mapping for a runtime.Object-only type",
			targets: []any{
				struct {
					DesiredComposite *fakeRuntimeObject `crossplane:"desired-composite"`
				}{
					DesiredComposite: &fakeRuntimeObject{Foo: "bar"},
				},
			},
			dsdc: map[resource.Name]*resource.DesiredComposed{},
			xr:   NewComposite(t).WithObject(&fakeRuntimeObject{Foo: "bar"}).Build(),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *gt.T) {
			xr, dsdc, err := inventory.ExtractDesiredState(tc.targets...)

			testing.AssertEqual(t, tc.xr, xr, "Composite Resource")
			testing.AssertEqual(t, tc.dsdc, dsdc, "Composed Resource")

			if tc.err != nil {
				require.Error(t, err)
				require.EqualError(t, err, tc.err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}
