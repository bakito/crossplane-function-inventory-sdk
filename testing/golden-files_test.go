package testing

import (
	"context"
	"errors"
	gt "testing"

	"github.com/stretchr/testify/assert"

	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
)

// TestGoldenFileCase_RunError covers the error path of GoldenFileCase.Run, which returns
// before any golden file is read or written.
func TestGoldenFileCase_RunError(t *gt.T) {
	tests := map[string]struct {
		err        error
		returned   error
		shouldFail bool
	}{
		"matching error": {
			err:      errors.New("boom"),
			returned: errors.New("boom"),
		},
		"error differing only in the modal verb": {
			err:      errors.New(msgVerbCannot),
			returned: errors.New(msgVerbUnableTo),
		},
		"error matcher": {
			err:      ErrorContains("cannot marshal to JSON"),
			returned: errors.New(msgVerbUnableTo),
		},
		"different error": {
			err:        errors.New("boom"),
			returned:   errors.New("bang"),
			shouldFail: true,
		},
		"unexpected error": {
			returned:   errors.New("boom"),
			shouldFail: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *gt.T) {
			c := GoldenFileCase{
				Reason:          "Should verify the error returned by the function.",
				Composite:       "testdata/resource.yaml",
				DesiredLocation: t.TempDir(),
				Err:             tc.err,
			}
			mockService := &mockFunctionRunnerServiceServer{
				runFunctionFunc: func(
					_ context.Context,
					_ *fnv1.RunFunctionRequest,
				) (*fnv1.RunFunctionResponse, error) {
					return nil, tc.returned
				},
			}

			mt := &mockTesting{T: t, shouldFail: tc.shouldFail}
			c.Run(mt, name, mockService)

			assert.Equal(t, tc.shouldFail, mt.failed)
		})
	}
}
