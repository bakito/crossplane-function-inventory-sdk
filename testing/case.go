package testing

import (
	"context"
	"fmt"
	"sync"
	gt "testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/resource"
)

type Case struct {
	// Reason is a human-readable description of the test case.
	Reason string
	// Args are the arguments available in the request for the test case.
	Args Args
	// Want is the expected response for the test case.
	Want Want
	// HookFunction is an optional function that can be run before/after the test case is run.
	HookFunction func()
	// DiffOptions are the diff options for the test case.
	DiffOptions []DiffOption
	// Parallel run test casse n times parallel
	Parallel int
	// Results the expected results returned by the function. Empty of OK
	Results []FunctionResult
}

// Args function request arguments.
type Args struct {
	Ctx context.Context //nolint:containedctx // ok for this test case
	Req *fnv1.RunFunctionRequest
}

// Want function response and error.
type Want struct {
	Rsp *fnv1.RunFunctionResponse
	Err error
}

// FunctionResult is the result of a function execution.
type FunctionResult struct {
	Severity fnv1.Severity
	Message  string
}

// Run runs the function test case.
func (c *Case) Run(tb gt.TB, f fnv1.FunctionRunnerServiceServer) {
	tb.Helper()
	if c.Parallel > 1 {
		var wg sync.WaitGroup
		for range c.Parallel {
			wg.Go(func() {
				c.runCase(tb, f)
			})
		}
		wg.Wait()
	} else {
		c.runCase(tb, f)
	}
}

func (c *Case) runCase(tb gt.TB, f fnv1.FunctionRunnerServiceServer) {
	tb.Helper()
	d := newDiff(c.DiffOptions)
	rsp, err := f.RunFunction(c.Args.Ctx, c.Args.Req)

	if c.Want.Err != nil {
		require.Error(tb, err)
		require.EqualError(tb, err, c.Want.Err.Error())
		return
	}
	if c.Want.Rsp != nil {
		require.NotNil(tb, rsp)
	}

	verifyResponseResults(tb, rsp, c.Results)

	if c.Want.Rsp != nil {
		if d.granular {
			c.granularDiff(
				tb,
				d,
				c.Want.Rsp.GetDesired().GetComposite().GetResource(),
				rsp.GetDesired().GetComposite().GetResource(),
				"Desired Composite",
			)

			for name, res := range rsp.GetDesired().GetResources() {
				c.granularDiff(
					tb,
					d,
					c.Want.Rsp.GetDesired().GetResources()[name].GetResource(),
					res.GetResource(),
					fmt.Sprintf("Desired Resource %q", name),
				)
			}
		} else {
			diff, err := d.unifiedDiff(c.Want.Rsp, rsp, "Response")
			if err != nil {
				tb.Errorf("Error in diffing response: %v", err)
			} else if diff != "" {
				tb.Errorf(
					"%s\nf.RunFunction(...): -want rsp, +got rsp:\n%s",
					c.Reason,
					Colorize(diff),
				)
			}
		}
	}
}

func (c *Case) granularDiff(tb gt.TB, d *diff, want, got *structpb.Struct, desc string) {
	tb.Helper()
	g, err := toUnstructured(got)
	require.NoError(tb, err)
	w, err := toUnstructured(want)
	require.NoError(tb, err)

	diff, err := d.unifiedDiff(w, g, desc)
	if err != nil {
		tb.Errorf("Error in diffing %s: %v", desc, err)
	} else if diff != "" {
		tb.Errorf("%s\nf.RunFunction(...): -want rsp, +got rsp:\n%s", c.Reason, Colorize(diff))
	}
}

func toUnstructured(got *structpb.Struct) (unstructured.Unstructured, error) {
	us := &unstructured.Unstructured{}
	err := resource.AsObject(got, us)
	return *us, err
}

func verifyResponseResults(tb gt.TB, rsp *fnv1.RunFunctionResponse, results []FunctionResult) {
	tb.Helper()
	assert.Len(tb, rsp.GetResults(), len(results))
	if len(results) > 0 {
		for i, result := range results {
			assert.Equal(tb, result.Severity, rsp.GetResults()[i].GetSeverity())
			if result.Message != "" {
				assert.Equal(tb, result.Message, rsp.GetResults()[i].GetMessage())
			}
		}
	}
}
