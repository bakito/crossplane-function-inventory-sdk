package inventory

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/aymanbagabas/go-udiff"
	ft "github.com/bakito/crossplane-function-inventory-sdk/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"

	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
)

const (
	testKey      = "test"
	fooKey       = "foo"
	configMapKey = "cm-test"
	podMapKey    = "pod-test"

	testFileDir = "../testdata/inventory"
)

// run `go test -v ./inventory/ -update-inventory-tests` or `make test-update` to update expected test files.
var update = flag.Bool("update-inventory-tests", false, "update golden files")

func TestMappingFor(t *testing.T) {
	mapping := Mapping{
		"ns": func() (runtime.Object, error) { return &corev1.Namespace{}, nil },
		"cm": func() (runtime.Object, error) { return &corev1.ConfigMap{}, nil },
	}

	tests := []struct {
		name    string
		prefix  string
		wantNil bool
	}{
		{
			name:    "prefix exists",
			prefix:  "ns-test",
			wantNil: false,
		},
		{
			name:    "prefix does not exist",
			prefix:  "unknown-test",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mappingFor(mapping, tt.prefix)
			if tt.wantNil {
				assert.Nil(t, got)
			} else {
				assert.NotNil(t, got)
			}
		})
	}
}

func TestResourceMap(t *testing.T) {
	var (
		ns        = &corev1.Namespace{}
		inventory = ResourceMap{
			testKey: &Resource{ro: ns},
		}
	)

	t.Run("HasResource", func(t *testing.T) {
		tests := []struct {
			name string
			key  string
			want bool
		}{
			{
				name: "resource exists",
				key:  testKey,
				want: true,
			},
			{
				name: "resource does not exist",
				key:  "missing",
				want: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := HasResource(inventory, tt.key)
				assert.Equal(t, tt.want, got)
			})
		}
	})

	t.Run("GetResource", func(t *testing.T) {
		tests := []struct {
			name    string
			key     string
			wantOK  bool
			wantNil bool
		}{
			{
				name:    "resource exists",
				key:     testKey,
				wantOK:  true,
				wantNil: false,
			},
			{
				name:    "resource does not exist",
				key:     "missing",
				wantOK:  false,
				wantNil: true,
			},
			{
				name:    "resource wrong type",
				key:     testKey,
				wantOK:  false,
				wantNil: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var got any
				var ok bool

				if tt.name == "resource wrong type" {
					got, _, _, ok = GetResource[*corev1.ConfigMap](inventory, tt.key)
				} else {
					got, _, _, ok = GetResource[*corev1.Namespace](inventory, tt.key)
				}

				assert.Equal(t, tt.wantOK, ok)
				if tt.wantNil {
					assert.Nil(t, got)
				} else {
					assert.Equal(t, ns, got)
				}
			})
		}
	})
}

func TestResourcesMap(t *testing.T) {
	var (
		ns        = &corev1.Namespace{}
		inventory = ResourcesMap{
			testKey: {&Resource{ro: ns}},
		}
	)

	t.Run("CountResources", func(t *testing.T) {
		tests := []struct {
			name string
			key  string
			want int
		}{
			{
				name: "resources exist",
				key:  testKey,
				want: 1,
			},
			{
				name: "resources do not exist",
				key:  "missing",
				want: 0,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := CountResources(inventory, tt.key)
				assert.Equal(t, tt.want, got)
			})
		}
	})

	t.Run("GetResources", func(t *testing.T) {
		tests := []struct {
			name    string
			key     string
			wantOK  bool
			wantNil bool
		}{
			{
				name:    "resources exist",
				key:     testKey,
				wantOK:  true,
				wantNil: false,
			},
			{
				name:    "resources do not exist",
				key:     "missing",
				wantOK:  false,
				wantNil: true,
			},
			{
				name:    "resource wrong type",
				key:     testKey,
				wantOK:  false,
				wantNil: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var got any
				var ok bool
				var ready []fnv1.Ready
				var expLen int

				if tt.name == "resource wrong type" {
					var cmSlice []*corev1.ConfigMap
					cmSlice, ready, _, ok = GetResources[*corev1.ConfigMap](inventory, tt.key)
					expLen = len(cmSlice)
					got = cmSlice
				} else {
					var nsSlice []*corev1.Namespace
					nsSlice, ready, _, ok = GetResources[*corev1.Namespace](inventory, tt.key)
					expLen = len(nsSlice)
					got = nsSlice
				}

				assert.Equal(t, tt.wantOK, ok)
				if tt.wantNil {
					assert.Nil(t, got)
				} else {
					assert.Equal(t, []*corev1.Namespace{ns}, got)
				}
				assert.Len(t, ready, expLen)
			})
		}
	})
}

func TestBuildInventory(t *testing.T) {
	log := logging.NewNopLogger()
	mapping := Mapping{
		"ns": func() (runtime.Object, error) { return &corev1.Namespace{}, nil },
		"cm": func() (runtime.Object, error) { return &corev1.ConfigMap{}, nil },
	}

	tests := []struct {
		name    string
		req     *fnv1.RunFunctionRequest
		wantErr bool
		checkFn func(*testing.T, Reader)
	}{
		{
			name: "succeeds with composite undefined",
			req: &fnv1.RunFunctionRequest{
				Desired: &fnv1.State{
					Resources: map[string]*fnv1.Resource{
						configMapKey: {Resource: &structpb.Struct{}},
					},
				},
				Observed: &fnv1.State{
					Resources: map[string]*fnv1.Resource{
						"ns-test": {Resource: &structpb.Struct{}},
					},
				},
				RequiredResources: map[string]*fnv1.Resources{
					"ns-required": {Items: []*fnv1.Resource{{Resource: &structpb.Struct{}}}},
				},
			},
			checkFn: func(t *testing.T, r Reader) {
				t.Helper()
				assert.Nil(t, r.GetDesiredComposite())
				assert.Len(t, r.GetDesiredComposed(), 1)
				assert.Nil(t, r.GetObservedComposite())
				assert.Len(t, r.GetObservedComposed(), 1)
				assert.Len(t, r.GetRequirements(), 1)
			},
		},
		{
			name: "succeeds with composite defined",
			req: &fnv1.RunFunctionRequest{
				Desired: &fnv1.State{
					Composite: &fnv1.Resource{Resource: &structpb.Struct{}},
					Resources: map[string]*fnv1.Resource{
						configMapKey: {Resource: &structpb.Struct{}},
					},
				},
				Observed: &fnv1.State{
					Composite: &fnv1.Resource{Resource: &structpb.Struct{}},
					Resources: map[string]*fnv1.Resource{
						"ns-test": {Resource: &structpb.Struct{}},
					},
				},
				RequiredResources: map[string]*fnv1.Resources{
					"ns-required": {Items: []*fnv1.Resource{{Resource: &structpb.Struct{}}}},
				},
			},
			checkFn: func(t *testing.T, r Reader) {
				t.Helper()
				assert.NotNil(t, r.GetDesiredComposite())
				assert.Len(t, r.GetDesiredComposed(), 1)
				assert.NotNil(t, r.GetObservedComposite())
				assert.Len(t, r.GetObservedComposed(), 1)
				assert.Len(t, r.GetRequirements(), 1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inv, err := BuildInventory(log, tt.req,
				WithComposite(func() (runtime.Object, error) { return &corev1.Secret{}, nil }),
				WithInput(func() (runtime.Object, error) { return &corev1.Service{}, nil }),
				WithMapping(mapping),
			)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, inv)

			if tt.checkFn != nil {
				tt.checkFn(t, inv)
			}
		})
	}
}

func TestBuildInventoryGracefulConversion(t *testing.T) {
	log := logging.NewNopLogger()
	mapping := Mapping{
		"pod": func() (runtime.Object, error) { return &corev1.Pod{}, nil },
	}

	tests := []struct {
		name             string
		req              *fnv1.RunFunctionRequest
		wantErr          bool
		graceful         bool
		conversionErrors int
		checkFn          func(*testing.T, *corev1.Pod)
	}{
		{
			name: "success with valid json data",
			req: &fnv1.RunFunctionRequest{
				Observed: &fnv1.State{
					Composite: &fnv1.Resource{Resource: &structpb.Struct{}},
					Resources: map[string]*fnv1.Resource{
						podMapKey: {
							Resource: ft.From(filepath.Join(testFileDir, "pod-valid.yaml")).
								ToStruct(),
						},
					},
				},
			},
			checkFn: func(t *testing.T, pod *corev1.Pod) {
				t.Helper()
				assert.NotNil(t, pod)
				assert.Equal(t, "liveness", pod.Spec.Containers[0].Args[0])
			},
		},
		{
			name: "fails with invalid json data",
			req: &fnv1.RunFunctionRequest{
				Observed: &fnv1.State{
					Composite: &fnv1.Resource{Resource: &structpb.Struct{}},
					Resources: map[string]*fnv1.Resource{
						podMapKey: {
							Resource: ft.From(filepath.Join(testFileDir, "pod-invalid.yaml")).
								ToStruct(),
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "success when graceful is conversion is enabled",
			req: &fnv1.RunFunctionRequest{
				Observed: &fnv1.State{
					Composite: &fnv1.Resource{Resource: &structpb.Struct{}},
					Resources: map[string]*fnv1.Resource{
						podMapKey: {
							Resource: ft.From(filepath.Join(testFileDir, "pod-invalid.yaml")).
								ToStruct(),
						},
					},
				},
			},
			checkFn: func(t *testing.T, pod *corev1.Pod) {
				t.Helper()
				assert.NotNil(t, pod)
				assert.Equal(t, "liveness", pod.Spec.Containers[0].Args[0])
			},
			graceful:         true,
			conversionErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := []Option{
				WithComposite(func() (runtime.Object, error) { return &corev1.Secret{}, nil }),
				WithInput(func() (runtime.Object, error) { return &corev1.Service{}, nil }),
				WithMapping(mapping),
			}
			var inv Reader
			var err error
			var conversionErrors []string
			if tt.graceful {
				inv, conversionErrors, err = BuildGracefulInventory(log, tt.req, true, opts...)
			} else {
				inv, err = BuildInventory(log, tt.req, opts...)
			}

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, conversionErrors, tt.conversionErrors)
			assert.NotNil(t, inv)

			if tt.checkFn != nil {
				composed := inv.GetObservedComposed()
				assert.Len(t, composed, 1)
				cmObj := composed[podMapKey]
				pod, ok := cmObj.ro.(*corev1.Pod)
				require.True(t, ok)
				tt.checkFn(t, pod)
			}
		})
	}
}

// TestBuildInventoryGracefulDiff verifies that only the properties not known are not marshaled.
func TestBuildInventoryGracefulDiff(t *testing.T) {
	log := logging.NewNopLogger()
	mapping := Mapping{
		"pod": func() (runtime.Object, error) { return &corev1.Pod{}, nil },
	}

	tests := []struct {
		name         string
		inputFile    string
		expectedFile string
	}{
		{
			name:         "verify expected with valid pod file",
			inputFile:    "pod-valid.yaml",
			expectedFile: "expected-pod-valid.yaml",
		},
		{
			name:         "verify expected with invalid pod file",
			inputFile:    "pod-invalid.yaml",
			expectedFile: "expected-pod-invalid.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := []Option{
				WithComposite(func() (runtime.Object, error) { return &corev1.Secret{}, nil }),
				WithInput(func() (runtime.Object, error) { return &corev1.Service{}, nil }),
				WithMapping(mapping),
			}

			req := &fnv1.RunFunctionRequest{
				Observed: &fnv1.State{
					Composite: &fnv1.Resource{Resource: &structpb.Struct{}},
					Resources: map[string]*fnv1.Resource{
						podMapKey: {
							Resource: ft.From(filepath.Join(testFileDir, tt.inputFile)).ToStruct(),
						},
					},
				},
			}

			inv, _, err := BuildGracefulInventory(log, req, true, opts...)

			require.NoError(t, err)
			assert.NotNil(t, inv)

			composed := inv.GetObservedComposed()
			assert.Len(t, composed, 1)
			cmObj := composed[podMapKey]
			pod, ok := cmObj.ro.(*corev1.Pod)
			require.True(t, ok)
			actual, err := yaml.Marshal(pod)
			require.NoError(t, err)

			if *update {
				err := os.WriteFile(filepath.Join(testFileDir, tt.expectedFile), actual, 0o600)
				require.NoError(t, err)
				return
			}

			expected, err := os.ReadFile(filepath.Join(testFileDir, tt.expectedFile))
			require.NoError(t, err)
			diff, err := unifiedDiff(string(expected), string(actual), "expected Pod", "actual Pod")
			if err != nil {
				t.Errorf("Error in diffing Pod: %v", err)
			} else if diff != "" {
				t.Errorf("%s\nPod: -want rsp, +got rsp:\n%s", tt.name, ft.Colorize(diff))
			}
		})
	}
}

func TestFindConflictingPrefix(t *testing.T) {
	tests := []struct {
		name     string
		existing []string
		test     string
		want     string
	}{
		{
			name: "no conflict in empty map",
			test: fooKey,
			want: "",
		},
		{
			name:     "no conflict with same key",
			existing: []string{fooKey},
			test:     fooKey,
			want:     "",
		},
		{
			name:     "conflict with same prefix",
			existing: []string{fooKey},
			test:     "foo-bar",
			want:     fooKey,
		},
		{
			name:     "conflict with same prefix reversed",
			existing: []string{"foo-bar"},
			test:     fooKey,
			want:     "foo-bar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := make(map[string]bool)
			for _, e := range tt.existing {
				m[e] = false
			}
			got := FindConflictingPrefix(m, tt.test)
			assert.Equal(t, tt.want, got)
		})
	}
}

func unifiedDiff(expected, actual, descExpected, descActual string) (string, error) {
	edits := udiff.Strings(expected, actual)
	return udiff.ToUnified(
		"Expected "+descExpected,
		"Actual "+descActual,
		expected,
		edits,
		5,
	)
}
