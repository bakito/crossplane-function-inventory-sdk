package require

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const testGroup = "test-group"

type mockObject struct{}

func (m *mockObject) GetObjectKind() schema.ObjectKind {
	return schema.EmptyObjectKind
}

func (m *mockObject) DeepCopyObject() runtime.Object {
	return new(*m)
}

func TestFindGVK(t *testing.T) {
	tests := []struct {
		name        string
		setupScheme func(scheme *runtime.Scheme)
		version     string
		wantGVK     schema.GroupVersionKind
		wantErr     string
	}{
		{
			name:        "no object kinds found",
			setupScheme: func(scheme *runtime.Scheme) {},
			version:     "",
			wantGVK:     schema.GroupVersionKind{},
			wantErr:     "no kind is registered for the type",
		},
		{
			name: "multiple versions without version specified",
			setupScheme: func(scheme *runtime.Scheme) {
				scheme.AddKnownTypes(
					schema.GroupVersion{Group: testGroup, Version: "v1"},
					&mockObject{},
				)
				scheme.AddKnownTypes(
					schema.GroupVersion{Group: testGroup, Version: "v2"},
					&mockObject{},
				)
			},
			version: "",
			wantGVK: schema.GroupVersionKind{},
			wantErr: "version required for multiple object kinds",
		},
		{
			name: "single object kind exists",
			setupScheme: func(scheme *runtime.Scheme) {
				scheme.AddKnownTypes(
					schema.GroupVersion{Group: testGroup, Version: "v1"},
					&mockObject{},
				)
			},
			version: "",
			wantGVK: schema.GroupVersionKind{
				Group:   testGroup,
				Version: "v1",
				Kind:    "mockObject",
			},
			wantErr: "",
		},
		{
			name: "no matching version found",
			setupScheme: func(scheme *runtime.Scheme) {
				scheme.AddKnownTypes(
					schema.GroupVersion{Group: testGroup, Version: "v1"},
					&mockObject{},
				)
			},
			version: "v2",
			wantGVK: schema.GroupVersionKind{},
			wantErr: "no object kind found with version v2",
		},
		{
			name: "multiple kinds with matching version",
			setupScheme: func(scheme *runtime.Scheme) {
				scheme.AddKnownTypes(
					schema.GroupVersion{Group: testGroup, Version: "v1"},
					&mockObject{},
				)
				scheme.AddKnownTypes(
					schema.GroupVersion{Group: testGroup, Version: "v2"},
					&mockObject{},
				)
			},
			version: "v2",
			wantGVK: schema.GroupVersionKind{
				Group:   testGroup,
				Version: "v2",
				Kind:    "mockObject",
			},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			tt.setupScheme(scheme)

			gotGVK, err := findGVK(&mockObject{}, scheme, tt.version)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantGVK, gotGVK)
			}
		})
	}
}
