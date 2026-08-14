package condition

import (
	"fmt"
	"strings"

	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/response"
)

const (
	maxMessageLength = 500

	TypeResourceConversion = "ResourceConversion"

	ReasonResourceConversionError      = "SemanticError"
	ReasonResourceConversionCompatible = "Compatible"
)

// ApplyJSONConversionCondition applies the ResourceConversion condition to the response.
func ApplyJSONConversionCondition(rsp *fnv1.RunFunctionResponse, conversionIssues []string) {
	if len(conversionIssues) > 0 {
		response.ConditionFalse(rsp, TypeResourceConversion, ReasonResourceConversionError).
			WithMessage(truncateMessage(fmt.Sprintf("%d conversion issues: %s", len(conversionIssues), strings.Join(conversionIssues, ", ")), maxMessageLength)).
			TargetComposite()
	} else {
		response.ConditionTrue(rsp, TypeResourceConversion, ReasonResourceConversionCompatible).
			TargetComposite()
	}
}

func truncateMessage(msg string, maxLength int) string {
	if len(msg) > maxLength {
		return msg[:maxLength] + "..."
	}
	return msg
}
