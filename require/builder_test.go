package require

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
)

const (
	testObjectName = "test-object"
	testKind       = "TestKind"
	testLabelKey   = "key"
	testLabelValue = "value"
)

func TestRequirementGet(t *testing.T) {
	tests := []struct {
		name         string
		requirement  *requirement
		wantName     string
		wantSelector *fnv1.ResourceSelector
		wantErr      bool
	}{
		{
			name: "with valid runtime object",
			requirement: &requirement{
				objectName: testObjectName,
				ro:         &corev1.Namespace{},
				scheme: func() *runtime.Scheme {
					s := runtime.NewScheme()
					_ = corev1.AddToScheme(s)
					return s
				}(),
			},
			wantName: testObjectName,
			wantSelector: &fnv1.ResourceSelector{
				ApiVersion: "v1",
				Kind:       "Namespace",
				Match:      &fnv1.ResourceSelector_MatchName{},
			},
			wantErr: false,
		},
		{
			name: "without runtime object",
			requirement: &requirement{
				objectName: testObjectName,
				apiVersion: "v1",
				kind:       testKind,
			},
			wantName: testObjectName,
			wantSelector: &fnv1.ResourceSelector{
				ApiVersion: "v1",
				Kind:       testKind,
				Match:      &fnv1.ResourceSelector_MatchName{},
			},
			wantErr: false,
		},
		{
			name: "with match labels",
			requirement: &requirement{
				objectName:  testObjectName,
				apiVersion:  "v1",
				kind:        testKind,
				matchLabels: map[string]string{testLabelKey: testLabelValue},
			},
			wantName: testObjectName,
			wantSelector: &fnv1.ResourceSelector{
				ApiVersion: "v1",
				Kind:       testKind,
				Match: &fnv1.ResourceSelector_MatchLabels{
					MatchLabels: &fnv1.MatchLabels{
						Labels: map[string]string{testLabelKey: testLabelValue},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "with match labels and namespace",
			requirement: &requirement{
				objectName:  testObjectName,
				apiVersion:  "v1",
				kind:        testKind,
				namespace:   new("srp-customer"),
				matchLabels: map[string]string{testLabelKey: testLabelValue},
			},
			wantName: testObjectName,
			wantSelector: &fnv1.ResourceSelector{
				ApiVersion: "v1",
				Kind:       testKind,
				Namespace:  new("srp-customer"),
				Match: &fnv1.ResourceSelector_MatchLabels{
					MatchLabels: &fnv1.MatchLabels{
						Labels: map[string]string{testLabelKey: testLabelValue},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "with match name",
			requirement: &requirement{
				objectName: testObjectName,
				apiVersion: "v1",
				kind:       testKind,
				matchName:  "resource-name",
			},
			wantName: testObjectName,
			wantSelector: &fnv1.ResourceSelector{
				ApiVersion: "v1",
				Kind:       testKind,
				Match: &fnv1.ResourceSelector_MatchName{
					MatchName: "resource-name",
				},
			},
			wantErr: false,
		},
		{
			name: "with match name and namespace",
			requirement: &requirement{
				objectName: testObjectName,
				apiVersion: "v1",
				kind:       testKind,
				namespace:  new("testnamespace"),
				matchName:  "resource-name",
			},
			wantName: testObjectName,
			wantSelector: &fnv1.ResourceSelector{
				ApiVersion: "v1",
				Kind:       testKind,
				Namespace:  new("testnamespace"),
				Match: &fnv1.ResourceSelector_MatchName{
					MatchName: "resource-name",
				},
			},
			wantErr: false,
		},
		{
			name: "when runtime object is nil",
			requirement: &requirement{
				objectName: testObjectName,
				ro:         nil,
			},
			wantName:     "",
			wantSelector: nil,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, selector, err := tt.requirement.Get()

			if tt.wantErr {
				require.Error(t, err)
				assert.Empty(t, name)
				assert.Nil(t, selector)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantName, name)
			assert.Equal(t, tt.wantSelector.GetApiVersion(), selector.GetApiVersion())
			assert.Equal(t, tt.wantSelector.GetKind(), selector.GetKind())

			if ml, ok := tt.wantSelector.GetMatch().(*fnv1.ResourceSelector_MatchLabels); ok {
				actualML, ok := selector.GetMatch().(*fnv1.ResourceSelector_MatchLabels)
				assert.True(t, ok)
				assert.Equal(t, ml.MatchLabels.GetLabels(), actualML.MatchLabels.GetLabels())
			}

			if mn, ok := tt.wantSelector.GetMatch().(*fnv1.ResourceSelector_MatchName); ok {
				actualMN, ok := selector.GetMatch().(*fnv1.ResourceSelector_MatchName)
				assert.True(t, ok)
				assert.Equal(t, mn.MatchName, actualMN.MatchName)
			}
		})
	}
}
