package testing

import (
	"context"
	"errors"
	"testing"

	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
)

const (
	testInput  = "test input"
	testOutput = "test output"
)

func TestCase_Run(t *testing.T) {
	tests := []struct {
		name       string
		c          Case
		mock       func(ctx context.Context, req *fnv1.RunFunctionRequest) (*fnv1.RunFunctionResponse, error)
		shouldFail bool
	}{
		{
			name: "success",
			c: Case{
				Reason: "Should return the expected response without error.",
				Args: Args{
					Ctx: context.Background(),
					Req: &fnv1.RunFunctionRequest{Meta: &fnv1.RequestMeta{Tag: testInput}},
				},
				Want: Want{
					Rsp: &fnv1.RunFunctionResponse{Meta: &fnv1.ResponseMeta{Tag: testOutput}},
					Err: nil,
				},
			},
			mock: func(ctx context.Context, req *fnv1.RunFunctionRequest) (*fnv1.RunFunctionResponse, error) {
				return &fnv1.RunFunctionResponse{Meta: &fnv1.ResponseMeta{Tag: testOutput}}, nil
			},
		},
		{
			name: "function error",
			c: Case{
				Reason: "Should return an error when the function fails.",
				Args: Args{
					Ctx: context.Background(),
					Req: &fnv1.RunFunctionRequest{Meta: &fnv1.RequestMeta{Tag: testInput}},
				},
				Want: Want{
					Rsp: nil,
					Err: errors.New("function error"),
				},
			},
			mock: func(ctx context.Context, req *fnv1.RunFunctionRequest) (*fnv1.RunFunctionResponse, error) {
				return nil, errors.New("function error")
			},
		},
		{
			name: "unexpected function error",
			c: Case{
				Reason: "Should return an error when the function fails.",
				Args: Args{
					Ctx: context.Background(),
					Req: &fnv1.RunFunctionRequest{Meta: &fnv1.RequestMeta{Tag: testInput}},
				},
				Want: Want{
					Rsp: nil,
					Err: errors.New("function error"),
				},
			},
			mock: func(ctx context.Context, req *fnv1.RunFunctionRequest) (*fnv1.RunFunctionResponse, error) {
				return nil, errors.New("unexpected function error")
			},
			shouldFail: true,
		},
		{
			name: "unexpected response",
			c: Case{
				Reason: "Should detect a mismatch in the response.",
				Args: Args{
					Ctx: context.Background(),
					Req: &fnv1.RunFunctionRequest{Meta: &fnv1.RequestMeta{Tag: testInput}},
				},
				Want: Want{
					Rsp: &fnv1.RunFunctionResponse{Meta: &fnv1.ResponseMeta{Tag: testOutput}},
					Err: nil,
				},
			},
			mock: func(ctx context.Context, req *fnv1.RunFunctionRequest) (*fnv1.RunFunctionResponse, error) {
				return &fnv1.RunFunctionResponse{
					Meta: &fnv1.ResponseMeta{Tag: "unexpected output"},
				}, nil
			},
			shouldFail: true,
		},
		{
			name: "nil response",
			c: Case{
				Reason: "Should detect a nil response when one is expected.",
				Args: Args{
					Ctx: context.Background(),
					Req: &fnv1.RunFunctionRequest{Meta: &fnv1.RequestMeta{Tag: testInput}},
				},
				Want: Want{
					Rsp: nil,
					Err: nil,
				},
			},
			mock: func(ctx context.Context, req *fnv1.RunFunctionRequest) (*fnv1.RunFunctionResponse, error) {
				return nil, nil
			},
		},
		{
			name: "default diff settings",
			c: Case{
				Reason: "Should apply default values for YAML indentation and diff context lines.",
				Args: Args{
					Ctx: context.Background(),
					Req: &fnv1.RunFunctionRequest{Meta: &fnv1.RequestMeta{Tag: testInput}},
				},
				Want: Want{
					Rsp: &fnv1.RunFunctionResponse{Meta: &fnv1.ResponseMeta{Tag: testOutput}},
					Err: nil,
				},
			},
			mock: func(ctx context.Context, req *fnv1.RunFunctionRequest) (*fnv1.RunFunctionResponse, error) {
				return &fnv1.RunFunctionResponse{Meta: &fnv1.ResponseMeta{Tag: testOutput}}, nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock FunctionRunnerServiceServer
			mockService := &mockFunctionRunnerServiceServer{
				runFunctionFunc: tt.mock,
			}

			mt := &mockTesting{T: t, shouldFail: tt.shouldFail}
			// Run the Case
			tt.c.Run(mt, mockService)

			if tt.shouldFail && !mt.failed {
				t.Error("Expected test to fail, but it passed.")
			}
		})
	}
}

// Mock implementation of FunctionRunnerServiceServer.
type mockFunctionRunnerServiceServer struct {
	fnv1.FunctionRunnerServiceServer
	runFunctionFunc func(ctx context.Context, req *fnv1.RunFunctionRequest) (*fnv1.RunFunctionResponse, error)
}

func (m *mockFunctionRunnerServiceServer) RunFunction(
	ctx context.Context,
	req *fnv1.RunFunctionRequest,
) (*fnv1.RunFunctionResponse, error) {
	if m.runFunctionFunc != nil {
		return m.runFunctionFunc(ctx, req)
	}
	return nil, nil
}

type mockTesting struct {
	*testing.T
	shouldFail bool
	failed     bool
}

func (mt *mockTesting) Errorf(format string, args ...any) {
	if mt.shouldFail {
		mt.failed = true
	} else {
		mt.T.Errorf(format, args...)
	}
}

func (mt *mockTesting) FailNow() {
	if mt.shouldFail {
		mt.failed = true
	} else {
		mt.T.FailNow()
	}
}
