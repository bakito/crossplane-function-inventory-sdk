package require

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
)

const testResource1 = "resource1"

func TestAssure(t *testing.T) {
	tests := []struct {
		name          string
		requirements  *Requirements
		request       *fnv1.RunFunctionRequest
		expectedProv  bool
		expectedError string
	}{
		{
			name: "NoSelectorsEmptyRequiredResources",
			requirements: &Requirements{
				selectors: map[string]*fnv1.ResourceSelector{},
			},
			request:       &fnv1.RunFunctionRequest{},
			expectedProv:  true,
			expectedError: "",
		},
		{
			name: "NoResourceDefinitions",
			requirements: &Requirements{
				selectors: map[string]*fnv1.ResourceSelector{
					testResource1: {
						Match: &fnv1.ResourceSelector_MatchName{
							MatchName: "match-name",
						},
					},
				},
			},
			request:       &fnv1.RunFunctionRequest{},
			expectedProv:  false,
			expectedError: "",
		},
		{
			name: "MissingRequiredResource",
			requirements: &Requirements{
				selectors: map[string]*fnv1.ResourceSelector{
					testResource1: {
						Match: &fnv1.ResourceSelector_MatchName{
							MatchName: "match-name",
						},
					},
				},
			},
			request: &fnv1.RunFunctionRequest{
				RequiredResources: map[string]*fnv1.Resources{},
			},
			expectedProv:  true,
			expectedError: "required resource \"resource1\" not found",
		},
		{
			name: "MatchNameValidationError",
			requirements: &Requirements{
				selectors: map[string]*fnv1.ResourceSelector{
					testResource1: {
						Match: &fnv1.ResourceSelector_MatchName{},
					},
				},
			},
			request: &fnv1.RunFunctionRequest{
				RequiredResources: map[string]*fnv1.Resources{
					testResource1: {
						Items: []*fnv1.Resource{},
					},
				},
			},
			expectedProv:  true,
			expectedError: "expected 1 required resource for name \"resource1\", but got 0",
		},
		{
			name: "MatchLabelsValidationError",
			requirements: &Requirements{
				selectors: map[string]*fnv1.ResourceSelector{
					testResource1: {
						Match: &fnv1.ResourceSelector_MatchLabels{
							MatchLabels: &fnv1.MatchLabels{
								Labels: map[string]string{testLabelKey: testLabelValue},
							},
						},
					},
				},
			},
			request: &fnv1.RunFunctionRequest{
				RequiredResources: map[string]*fnv1.Resources{
					testResource1: {
						Items: []*fnv1.Resource{},
					},
				},
			},
			expectedProv:  true,
			expectedError: "expected > 1 required resources for name \"resource1\", but got 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provided, err := tt.requirements.Assure(tt.request)
			if tt.expectedError != "" {
				require.EqualError(t, err, tt.expectedError)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.expectedProv, provided)
		})
	}
}

func TestRegister(t *testing.T) {
	tests := []struct {
		name                 string
		requirements         *Requirements
		response             *fnv1.RunFunctionResponse
		expectedRequirements *fnv1.Requirements
	}{
		{
			name: "EmptySelectors",
			requirements: &Requirements{
				selectors: map[string]*fnv1.ResourceSelector{},
			},
			response:             &fnv1.RunFunctionResponse{},
			expectedRequirements: nil,
		},
		{
			name: "NonEmptySelectors",
			requirements: &Requirements{
				selectors: map[string]*fnv1.ResourceSelector{
					testResource1: {
						Match: &fnv1.ResourceSelector_MatchLabels{
							MatchLabels: &fnv1.MatchLabels{
								Labels: map[string]string{testLabelKey: testLabelValue},
							},
						},
					},
				},
			},
			response: &fnv1.RunFunctionResponse{},
			expectedRequirements: &fnv1.Requirements{
				Resources: map[string]*fnv1.ResourceSelector{
					testResource1: {
						Match: &fnv1.ResourceSelector_MatchLabels{
							MatchLabels: &fnv1.MatchLabels{
								Labels: map[string]string{testLabelKey: testLabelValue},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.requirements.Register(tt.response)
			assert.Equal(t, tt.expectedRequirements, tt.response.GetRequirements())
		})
	}
}

func TestRequires(t *testing.T) {
	mockRequirement := func(name string, sel *fnv1.ResourceSelector, err error) Requirement {
		return &mockReq{name: name, selector: sel, err: err}
	}

	tests := []struct {
		name         string
		requirements []Requirement
		expected     *Requirements
		expectedErr  string
	}{
		{
			name:         "NoRequirements",
			requirements: nil,
			expected: &Requirements{
				selectors: map[string]*fnv1.ResourceSelector{},
			},
			expectedErr: "",
		},
		{
			name: "SingleRequirement",
			requirements: []Requirement{
				mockRequirement(testResource1, &fnv1.ResourceSelector{
					Match: &fnv1.ResourceSelector_MatchName{
						MatchName: "match",
					},
				}, nil),
			},
			expected: &Requirements{
				selectors: map[string]*fnv1.ResourceSelector{
					testResource1: {
						Match: &fnv1.ResourceSelector_MatchName{
							MatchName: "match",
						},
					},
				},
			},
			expectedErr: "",
		},
		{
			name: "RequirementWithError",
			requirements: []Requirement{
				mockRequirement(testResource1, nil, errors.New("error evaluating requirement")),
			},
			expectedErr: "error evaluating requirement",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Requires(tt.requirements...)
			if tt.expectedErr != "" {
				require.EqualError(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

type mockReq struct {
	name     string
	selector *fnv1.ResourceSelector
	err      error
}

func (m *mockReq) Get() (string, *fnv1.ResourceSelector, error) {
	return m.name, m.selector, m.err
}
