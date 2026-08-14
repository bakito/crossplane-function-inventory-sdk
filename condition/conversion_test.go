package condition

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
)

func TestApplyJSONConversionCondition(t *testing.T) {
	tests := []struct {
		name             string
		conversionIssues []string
		expectedType     string
		expectedStatus   fnv1.Status
		expectedReason   string
		expectedMessage  *string
	}{
		{
			name:             "conversion issues exist",
			conversionIssues: []string{"issue1", "issue2"},
			expectedType:     TypeResourceConversion,
			expectedReason:   ReasonResourceConversionError,
			expectedStatus:   fnv1.Status_STATUS_CONDITION_FALSE,
			expectedMessage:  new("2 conversion issues: issue1, issue2"),
		},
		{
			name:             "no conversion issues",
			conversionIssues: nil,
			expectedType:     TypeResourceConversion,
			expectedReason:   ReasonResourceConversionCompatible,
			expectedStatus:   fnv1.Status_STATUS_CONDITION_TRUE,
			expectedMessage:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rsp := &fnv1.RunFunctionResponse{}
			ApplyJSONConversionCondition(rsp, tt.conversionIssues)

			assert.Equal(t, tt.expectedType, rsp.GetConditions()[0].GetType())
			assert.Equal(t, tt.expectedStatus, rsp.GetConditions()[0].GetStatus())
			assert.Equal(t, tt.expectedReason, rsp.GetConditions()[0].GetReason())
			assert.Equal(t, tt.expectedMessage, rsp.GetConditions()[0].Message)
		})
	}
}

func TestTruncateMessage(t *testing.T) {
	tests := []struct {
		name      string
		msg       string
		maxLength int
		expected  string
	}{
		{
			name:      "message shorter than max length",
			msg:       "short message",
			maxLength: 100,
			expected:  "short message",
		},
		{
			name:      "message equal to max length",
			msg:       strings.Repeat("a", 100),
			maxLength: 100,
			expected:  strings.Repeat("a", 100),
		},
		{
			name:      "message longer than max length",
			msg:       strings.Repeat("b", 200),
			maxLength: 100,
			expected:  strings.Repeat("b", 100) + "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateMessage(tt.msg, tt.maxLength)
			assert.Equal(t, tt.expected, result)
		})
	}
}
