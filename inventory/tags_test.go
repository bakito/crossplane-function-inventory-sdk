package inventory_test

import (
	gt "testing"

	"github.com/bakito/crossplane-function-inventory-sdk/inventory"
	"github.com/bakito/crossplane-function-inventory-sdk/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
)

func TestTags_Inject(t *gt.T) {
	var (
		req *fnv1.RunFunctionRequest
		log logging.Logger
	)

	setup := func(t *gt.T) {
		t.Helper()
		log = logging.NewNopLogger()
		req = &fnv1.RunFunctionRequest{
			Desired: &fnv1.State{
				Resources: map[string]*fnv1.Resource{
					"cm-test": {
						Resource:          resourceFrom("cm.yaml"),
						Ready:             fnv1.Ready_READY_TRUE,
						ConnectionDetails: inventory.ConnectionDetails{"cm-conn": []byte("cm")},
					},
				},
				Composite: &fnv1.Resource{
					Resource: resourceFrom("sec.yaml"),
					Ready:    fnv1.Ready_READY_TRUE,
					ConnectionDetails: inventory.ConnectionDetails{
						"sec-conn": []byte("sec-desired"),
					},
				},
			},
			Observed: &fnv1.State{
				Resources: map[string]*fnv1.Resource{
					"node-test": {
						Resource:          resourceFrom("node.yaml"),
						Ready:             fnv1.Ready_READY_TRUE,
						ConnectionDetails: inventory.ConnectionDetails{"node-conn": []byte("node")},
					},
					"event-test": {
						Resource: resourceFrom("event.yaml"),
						Ready:    fnv1.Ready_READY_TRUE,
						ConnectionDetails: inventory.ConnectionDetails{
							"event-conn": []byte("event"),
						},
					},
				},
				Composite: &fnv1.Resource{
					Resource: resourceFrom("sec.yaml"),
					Ready:    fnv1.Ready_READY_TRUE,
					ConnectionDetails: inventory.ConnectionDetails{
						"sec-conn": []byte("sec-observed"),
					},
				},
			},
			RequiredResources: map[string]*fnv1.Resources{
				"pod-required": {
					Items: []*fnv1.Resource{
						{
							Resource: resourceFrom("pod.yaml"),
							Ready:    fnv1.Ready_READY_TRUE,
							ConnectionDetails: inventory.ConnectionDetails{
								"pod-conn": []byte("pod"),
							},
						},
					},
				},
				"svc-required": {
					Items: []*fnv1.Resource{
						{
							Resource: resourceFrom("svc.yaml"),
							Ready:    fnv1.Ready_READY_TRUE,
							ConnectionDetails: inventory.ConnectionDetails{
								"svc-ex1-conn": []byte("svc-ex1"),
							},
						},
						{
							Resource: resourceFrom("svc.yaml"),
							Ready:    fnv1.Ready_READY_TRUE,
							ConnectionDetails: inventory.ConnectionDetails{
								"svc-ex2-conn": []byte("svc-ex2"),
							},
						},
					},
				},
			},
			Input: &structpb.Struct{},
		}
	}

	buildInventory := func(t *gt.T) inventory.Reader {
		t.Helper()
		mapping := inventory.Mapping{
			"node": func() (runtime.Object, error) { return &corev1.Node{}, nil },
			"cm":   func() (runtime.Object, error) { return &corev1.ConfigMap{}, nil },
			"pod":  func() (runtime.Object, error) { return &corev1.Pod{}, nil },
			"svc":  func() (runtime.Object, error) { return &corev1.Service{}, nil },
		}

		inv, err := inventory.BuildInventory(
			log,
			req,
			inventory.WithComposite(
				func() (runtime.Object, error) { return &corev1.Secret{}, nil },
			),
			inventory.WithInput(func() (runtime.Object, error) { return &corev1.Pod{}, nil }),
			inventory.WithMapping(mapping),
		)
		require.NoError(t, err)
		return inv
	}

	t.Run("reader inject values field", func(t *gt.T) {
		setup(t)
		inv := buildInventory(t)

		type TestStruct struct {
			Reader inventory.Reader
		}
		ts := &TestStruct{}
		err := inventory.Inject(inv, ts)
		require.NoError(t, err)
		assert.NotNil(t, ts.Reader)
	})

	t.Run("inject values into input tagged field", func(t *gt.T) {
		setup(t)
		inv := buildInventory(t)

		type TestStruct struct {
			Input *corev1.Pod `crossplane:"input"`
		}
		ts := &TestStruct{}
		err := inventory.Inject(inv, ts)
		require.NoError(t, err)
		assert.NotNil(t, ts.Input)
	})

	t.Run("inject values into observed-composed tagged fields", func(t *gt.T) {
		setup(t)
		inv := buildInventory(t)

		type TestStruct struct {
			ObservedComposed      *corev1.Node                `crossplane:"observed-composed:node-test"`
			ObservedComposedReady fnv1.Ready                  `crossplane:"observed-composed:node-test"`
			ObservedComposedConn  inventory.ConnectionDetails `crossplane:"observed-composed:node-test"`
		}
		ts := &TestStruct{}
		err := inventory.Inject(inv, ts)
		require.NoError(t, err)

		assert.NotNil(t, ts.ObservedComposed)
		assert.Equal(t, fnv1.Ready_READY_TRUE, ts.ObservedComposedReady)
		assert.Equal(t, []byte("node"), ts.ObservedComposedConn["node-conn"])
	})

	t.Run("inject values into observed-composed tagged map fields", func(t *gt.T) {
		setup(t)
		inv := buildInventory(t)

		type TestStruct struct {
			ObservedComposedMap      map[string]*corev1.Node                `crossplane:"observed-composed:node-"`
			ObservedComposedMapReady map[string]fnv1.Ready                  `crossplane:"observed-composed:node-"`
			ObservedComposedMapConn  map[string]inventory.ConnectionDetails `crossplane:"observed-composed:node-"`
		}
		ts := &TestStruct{}
		err := inventory.Inject(inv, ts)
		require.NoError(t, err)

		require.NotNil(t, ts.ObservedComposedMap)
		assert.Len(t, ts.ObservedComposedMap, 1)
		_, ok := ts.ObservedComposedMap["node-test"]
		assert.True(t, ok)
		assert.Len(t, ts.ObservedComposedMapReady, len(ts.ObservedComposedMap))
		assert.Equal(t, fnv1.Ready_READY_TRUE, ts.ObservedComposedMapReady["node-test"])
		assert.Len(t, ts.ObservedComposedMapConn, len(ts.ObservedComposedMap))
		assert.Equal(t, []byte("node"), ts.ObservedComposedMapConn["node-test"]["node-conn"])
	})

	t.Run("inject values into desired-composed tagged fields", func(t *gt.T) {
		setup(t)
		inv := buildInventory(t)

		type TestStruct struct {
			DesiredComposed      *corev1.ConfigMap           `crossplane:"desired-composed:cm-test"`
			DesiredComposedReady fnv1.Ready                  `crossplane:"desired-composed:cm-test"`
			DesiredComposedConn  inventory.ConnectionDetails `crossplane:"desired-composed:cm-test"`
		}
		ts := &TestStruct{}
		err := inventory.Inject(inv, ts)
		require.NoError(t, err)
		assert.NotNil(t, ts.DesiredComposed)
		assert.Equal(t, fnv1.Ready_READY_TRUE, ts.DesiredComposedReady)
		assert.Equal(t, []byte("cm"), ts.DesiredComposedConn["cm-conn"])
	})

	t.Run("inject values into desired-composed tagged map fields", func(t *gt.T) {
		setup(t)
		inv := buildInventory(t)

		type TestStruct struct {
			Input                   *corev1.Pod                            `crossplane:"input"`
			DesiredComposedMap      map[string]*corev1.ConfigMap           `crossplane:"desired-composed:cm-"`
			DesiredComposedMapReady map[string]fnv1.Ready                  `crossplane:"desired-composed:cm-"`
			DesiredComposedMapConn  map[string]inventory.ConnectionDetails `crossplane:"desired-composed:cm-"`
		}
		ts := &TestStruct{}
		err := inventory.Inject(inv, ts)
		require.NoError(t, err)

		require.NotNil(t, ts.DesiredComposedMap)
		assert.Len(t, ts.DesiredComposedMap, 1)
		_, ok := ts.DesiredComposedMap["cm-test"]
		assert.True(t, ok)
		assert.Len(t, ts.DesiredComposedMapReady, len(ts.DesiredComposedMap))
		assert.Equal(t, fnv1.Ready_READY_TRUE, ts.DesiredComposedMapReady["cm-test"])
		assert.Len(t, ts.DesiredComposedMapConn, len(ts.DesiredComposedMap))
		assert.Equal(t, []byte("cm"), ts.DesiredComposedMapConn["cm-test"]["cm-conn"])
	})

	t.Run("inject values into observed-composite tagged fields", func(t *gt.T) {
		setup(t)
		inv := buildInventory(t)

		type TestStruct struct {
			ObservedComposite      *corev1.Secret              `crossplane:"observed-composite"`
			ObservedCompositeReady fnv1.Ready                  `crossplane:"observed-composite"`
			ObservedCompositeConn  inventory.ConnectionDetails `crossplane:"observed-composite"`
		}
		ts := &TestStruct{}
		err := inventory.Inject(inv, ts)
		require.NoError(t, err)

		assert.NotNil(t, ts.ObservedComposite)
		assert.Equal(t, fnv1.Ready_READY_TRUE, ts.ObservedCompositeReady)
		assert.Equal(t, []byte("sec-observed"), ts.ObservedCompositeConn["sec-conn"])
	})

	t.Run("inject values into desired-composite tagged fields", func(t *gt.T) {
		setup(t)
		inv := buildInventory(t)

		type TestStruct struct {
			DesiredComposite      *corev1.Secret              `crossplane:"desired-composite"`
			DesiredCompositeReady fnv1.Ready                  `crossplane:"desired-composite"`
			DesiredCompositeConn  inventory.ConnectionDetails `crossplane:"desired-composite"`
		}
		ts := &TestStruct{}
		err := inventory.Inject(inv, ts)
		require.NoError(t, err)

		assert.NotNil(t, ts.DesiredComposite)
		assert.Equal(t, fnv1.Ready_READY_TRUE, ts.DesiredCompositeReady)
		assert.Equal(t, []byte("sec-desired"), ts.DesiredCompositeConn["sec-conn"])
	})

	t.Run("inject values into required tagged fields", func(t *gt.T) {
		setup(t)
		inv := buildInventory(t)

		type TestStruct struct {
			RequiredPod      *corev1.Pod                 `crossplane:"required:pod-required"`
			RequiredPodReady fnv1.Ready                  `crossplane:"required:pod-required"`
			RequiredPodConn  inventory.ConnectionDetails `crossplane:"required:pod-required"`
		}
		ts := &TestStruct{}
		err := inventory.Inject(inv, ts)
		require.NoError(t, err)

		assert.NotNil(t, ts.RequiredPod)
		assert.Equal(t, fnv1.Ready_READY_TRUE, ts.RequiredPodReady)
		assert.Equal(t, []byte("pod"), ts.RequiredPodConn["pod-conn"])
	})

	t.Run("inject values into required tagged slice fields", func(t *gt.T) {
		setup(t)
		inv := buildInventory(t)

		type TestStruct struct {
			RequiredSvc      []*corev1.Service             `crossplane:"required:svc-required"`
			RequiredSvcReady []fnv1.Ready                  `crossplane:"required:svc-required"`
			RequiredSvcConn  []inventory.ConnectionDetails `crossplane:"required:svc-required"`
		}
		ts := &TestStruct{}
		err := inventory.Inject(inv, ts)
		require.NoError(t, err)

		require.Len(t, ts.RequiredSvc, 2)
		assert.Len(t, ts.RequiredSvcReady, len(ts.RequiredSvc))
		assert.ElementsMatch(
			t,
			[]fnv1.Ready{fnv1.Ready_READY_TRUE, fnv1.Ready_READY_TRUE},
			ts.RequiredSvcReady,
		)
		require.Len(t, ts.RequiredSvcConn, len(ts.RequiredSvc))
		assert.Equal(t, []byte("svc-ex1"), ts.RequiredSvcConn[0]["svc-ex1-conn"])
		assert.Equal(t, []byte("svc-ex2"), ts.RequiredSvcConn[1]["svc-ex2-conn"])
	})

	t.Run("fail for invalid tags configurations", func(t *gt.T) {
		setup(t)
		inv := buildInventory(t)

		tests := []struct {
			name          string
			testStruct    any
			expectedError string
		}{
			{
				name: "observed-composed: map with non-string key",
				testStruct: &struct {
					ObsMap map[int]*corev1.Node `crossplane:"observed-composed"`
				}{},
				expectedError: `tag observed-composed of field ObsMap has incorrect format - must be: "observed-composed:<resource-name-prefix>"`,
			},
			{
				name: "observed-composed: non-map without resource name",
				testStruct: &struct {
					ObsMap *corev1.Node `crossplane:"observed-composed"`
				}{},
				expectedError: `tag observed-composed of field ObsMap has incorrect format - must be: "observed-composed:<resource-name>"`,
			},
			{
				name: "observed-composed: to many tags",
				testStruct: &struct {
					ObsMap *corev1.Node `crossplane:"observed-composed:foo:bar"`
				}{},
				expectedError: `tag observed-composed:foo:bar of field ObsMap has incorrect format - must be: "observed-composed:<resource-name>"`,
			},
			{
				name: "input: to many tags",
				testStruct: &struct {
					ObsMap *corev1.Node `crossplane:"input:bar"`
				}{},
				expectedError: `tag input:bar of field ObsMap has incorrect format - must be: "input"`,
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *gt.T) {
				err := inventory.Inject(inv, tc.testStruct)
				require.Error(t, err)
				assert.Equal(t, tc.expectedError, err.Error())
			})
		}
	})

	t.Run("return error for nil target", func(t *gt.T) {
		setup(t)
		inv := buildInventory(t)
		err := inventory.Inject(inv, nil)
		require.EqualError(t, err, "target must be a non-nil pointer to struct")
	})

	t.Run("validateTarget function", func(t *gt.T) {
		setup(t)
		inv := buildInventory(t)

		t.Run("return error for zero-valued target", func(t *gt.T) {
			err := inventory.Inject(inv, struct{}{})
			require.EqualError(t, err, "target must be a non-nil pointer to struct")
		})

		t.Run("accept valid struct pointer", func(t *gt.T) {
			target := &struct{}{}
			inv := inventory.Reader(nil)
			err := inventory.Inject(inv, target)
			require.NoError(t, err)
		})
	})

	t.Run("return error for non-pointer target", func(t *gt.T) {
		setup(t)
		inv := buildInventory(t)
		var s struct{}
		err := inventory.Inject(inv, s)
		require.EqualError(t, err, "target must be a non-nil pointer to struct")
	})

	t.Run("return error for non-struct pointer target", func(t *gt.T) {
		setup(t)
		inv := buildInventory(t)
		var i int
		err := inventory.Inject(inv, &i)
		require.EqualError(t, err, "target must be a pointer to struct")
	})
}

func TestTags_BuildInventoryOptions(t *gt.T) {
	t.Run("return options for valid targets with consistent types", func(t *gt.T) {
		type CompositeStruct struct {
			Input     *corev1.Pod               `crossplane:"input"`
			Composite *corev1.Secret            `crossplane:"observed-composite"`
			Service   *corev1.Service           `crossplane:"required:svc"`
			ObsMapNs  map[string]*corev1.Node   `crossplane:"observed-composed:node-"`
			ObsMapSec map[string]*corev1.Secret `crossplane:"observed-composed:sec-"`
		}
		target := &CompositeStruct{}

		opts, err := inventory.BuildInventoryOptions(target)
		require.NoError(t, err)
		require.Len(t, opts, 3)

		// Build inventory with options and ensure injection works on the original scenario
		log := logging.NewNopLogger()
		req := &fnv1.RunFunctionRequest{
			Input: &structpb.Struct{},
			Observed: &fnv1.State{
				Composite: &fnv1.Resource{
					Resource: resourceFrom("sec.yaml"),
					Ready:    fnv1.Ready_READY_TRUE,
				},
			},
		}
		inv, err := inventory.BuildInventory(log, req, opts...)
		require.NoError(t, err)
		err = inventory.Inject(inv, target)
		require.NoError(t, err)
	})

	t.Run("error for inconsistent input types across targets", func(t *gt.T) {
		type TargetOne struct {
			Input *corev1.Pod `crossplane:"input"`
		}
		type TargetTwo struct {
			Input *corev1.Secret `crossplane:"input"`
		}
		_, err := inventory.BuildInventoryOptions(&TargetOne{}, &TargetTwo{})
		require.Error(t, err)
		assert.EqualError(t, err, "input type must be the same for all targets")
	})

	t.Run("error for inconsistent composite types across targets", func(t *gt.T) {
		type TargetOne struct {
			Composite *corev1.Pod `crossplane:"observed-composite"`
		}
		type TargetTwo struct {
			Composite *corev1.Secret `crossplane:"desired-composite"`
		}
		_, err := inventory.BuildInventoryOptions(&TargetOne{}, &TargetTwo{})
		require.Error(t, err)
		assert.EqualError(t, err, "desired-composite type must be the same for all targets")
	})

	t.Run("handle valid required and composed types", func(t *gt.T) {
		type TargetStruct struct {
			RequiredPod *corev1.Pod       `crossplane:"required:pod"`
			RequiredSvc []*corev1.Service `crossplane:"required:svc"`
		}
		target := &TargetStruct{}
		opts, err := inventory.BuildInventoryOptions(target)
		require.NoError(t, err)
		require.Len(t, opts, 1)
	})

	t.Run("error for inconsistent required types", func(t *gt.T) {
		type TargetOne struct {
			RequiredPod *corev1.Pod `crossplane:"required:svc"`
		}
		type TargetTwo struct {
			RequiredPod *corev1.Secret `crossplane:"required:svc"`
		}
		_, err := inventory.BuildInventoryOptions(&TargetOne{}, &TargetTwo{})
		require.Error(t, err)
		assert.EqualError(
			t,
			err,
			`required types svc must be the same for all targets. Got "*v1.Pod" and "*v1.Secret"`,
		)
	})

	t.Run("error for inconsistent prefix in mapped types", func(t *gt.T) {
		type TargetOne struct {
			RequiredPod map[string]*corev1.Pod `crossplane:"required:svc-"`
		}
		type TargetTwo struct {
			RequiredPod map[string]*corev1.Secret `crossplane:"observed-composed:svc-x-"`
		}
		_, err := inventory.BuildInventoryOptions(&TargetOne{}, &TargetTwo{})
		require.Error(t, err)
		assert.EqualError(
			t,
			err,
			`resource name prefix must be unique for tag types [required, observed-composed, desired-composed]. Got conflict with "svc-x-" and "svc-"`,
		)
	})

	t.Run("NOT error for consistent prefix in mapped types", func(t *gt.T) {
		type TargetOne struct {
			RequiredPod map[string]*corev1.Pod `crossplane:"required:svc-y"`
		}
		type TargetTwo struct {
			RequiredPod map[string]*corev1.Secret `crossplane:"observed-composed:svc-x-"`
		}
		_, err := inventory.BuildInventoryOptions(&TargetOne{}, &TargetTwo{})
		require.NoError(t, err)
	})
}

func resourceFrom(file string) *structpb.Struct {
	return testing.NewResourceFromFile("../testdata/" + file).ToStruct()
}
