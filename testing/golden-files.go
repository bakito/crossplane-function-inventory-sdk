package testing

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	gt "testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
	"gopkg.in/yaml.v3"

	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/request"
)

// Golden files test case.
// run `go test -v -update` to update the golden files

const (
	fileComposite = "composite"
	fileReady     = "ready"
)

var (
	update                = flag.Bool("update", false, "update golden files")
	invalidCaseNameCharRE = regexp.MustCompile(`[^a-zA-Z0-9_.-]+`)
)

type GoldenFileCase struct {
	// Reason is a human-readable description of the test case.
	Reason string
	// Composite the path ot the composite file to use for this test
	Composite string
	// Observed resources to mapped by their key used as observed resources in the function request
	Observed map[string]string
	// Required resources to mapped by their key used as required resources in the function request
	Required map[string]string
	// Input the function input file for the request
	Input string
	// DesiredLocation location of the desired files used to verif the output or the function
	DesiredLocation string
	// WithReady whether to verify the ready result of the desired resources in the function response.
	WithReady bool
	// Err the expected error returned by the function.
	Err error
	// Results the expected results returned by the function. Empty of OK
	Results []FunctionResult

	// DiffOptions are the diff options for the test case.
	DiffOptions []DiffOption

	diff     *diff
	xrName   string
	caseName string
}

// Run runs the function test case.
func (c *GoldenFileCase) Run(tb gt.TB, caseName string, f fnv1.FunctionRunnerServiceServer) {
	tb.Helper()

	c.caseName = sanitizeCaseName(caseName)

	require.NotEmpty(tb, c.Composite)
	require.NotEmpty(tb, c.DesiredLocation)
	info, err := os.Stat(c.DesiredLocation)
	require.NoError(tb, err, "DesiredLocation must exist")
	require.True(tb, info.IsDir(), "DesiredLocation must be a directory")

	c.diff = newDiff(c.DiffOptions)

	req := c.prepareRequest()

	xr, err := request.GetObservedCompositeResource(req)
	require.NoError(tb, err)

	c.xrName, err = xr.Resource.GetString("metadata.name")
	require.NoError(tb, err)

	desiredFiles, err := c.findDesiredFiles()
	require.NoError(tb, err)

	ctx := context.TODO()

	rsp, err := f.RunFunction(ctx, req)

	if c.Err != nil {
		require.Error(tb, err)
		require.EqualError(tb, err, c.Err.Error())
		return
	}
	require.NotNil(tb, rsp)

	verifyResponseResults(tb, rsp, c.Results)

	require.NoError(tb, err)

	c.verifyDesiredResources(tb, rsp, desiredFiles)
}

func (c *GoldenFileCase) verifyDesiredResources(
	tb gt.TB,
	rsp *fnv1.RunFunctionResponse,
	desiredFiles map[string]bool,
) {
	tb.Helper()
	if des := rsp.GetDesired().GetComposite().GetResource(); des != nil {
		file := c.verifyResource(tb, rsp.GetDesired().GetComposite().GetResource(), fileComposite)
		delete(desiredFiles, file)
	}

	currentReady := make(readyMap)
	for name, res := range rsp.GetDesired().GetResources() {
		file := c.verifyResource(tb, res.GetResource(), name)
		currentReady[name] = res.GetReady()
		delete(desiredFiles, file)
	}

	if c.WithReady {
		readyFileName := c.goldenFileName(fileReady)
		c.verifyReady(tb, currentReady, readyFileName)
		delete(desiredFiles, readyFileName)
	}

	if *update {
		for file := range desiredFiles {
			//nolint:testifylint // we want assert here to not abort the whole test
			assert.NoError(tb, os.Remove(file))
			tb.Logf("Deleted unused golden file %s", file)
		}
	} else {
		assert.Empty(tb, desiredFiles, "unused golden files for %s", c.Reason)
	}
}

func (c *GoldenFileCase) findDesiredFiles() (map[string]bool, error) {
	desiredFiles := make(map[string]bool)
	err := filepath.Walk(c.DesiredLocation, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasPrefix(info.Name(), c.goldenFilePrefix()) &&
			strings.HasSuffix(info.Name(), ".yaml") {
			desiredFiles[path] = true
		}
		return nil
	})
	return desiredFiles, err
}

func (c *GoldenFileCase) prepareRequest() *fnv1.RunFunctionRequest {
	rb := NewRequestBuilder().WithObservedComposite(From(c.Composite))
	if c.Input != "" {
		rb = rb.WithInput(From(c.Input))
	}

	for name, obs := range c.Observed {
		rb = rb.WithObservedResource(name, From(obs))
	}
	for name, req := range c.Required {
		rb = rb.WithRequiredResources(name, From(req).ToStruct())
	}

	req := rb.Build()
	return req
}

func (c *GoldenFileCase) verifyResource(tb gt.TB, resource *structpb.Struct, name string) string {
	tb.Helper()
	us, err := toUnstructured(resource)
	require.NoError(tb, err)
	got, err := c.diff.toYaml(us.Object)
	require.NoError(tb, err)

	var description string

	if name == "composite" {
		description = "Desired Composite"
	} else {
		description = "Desired Resource " + name
	}
	expectFile := c.goldenFileName(name)
	if *update {
		// #nosec G304 -- expectFile is constructed from trusted test data (DesiredLocation and xrName)
		// #nosec G306 -- we allow test files to be readable
		err = os.WriteFile(expectFile, []byte(got), 0o600)
		require.NoError(tb, err)
		tb.Logf("Updated golden file %s", expectFile)
		return expectFile
	}

	// #nosec G304 -- expectFile is constructed from trusted test data (DesiredLocation and xrName)
	want, err := os.ReadFile(expectFile)

	require.NoError(tb, err)
	diff, err := c.diff.unifiedYamlDiff(
		string(want),
		got,
		fmt.Sprintf("%s (%s)", description, expectFile),
		description,
	)
	if err != nil {
		tb.Errorf("Error in diffing %s: %v", description, err)
	} else if diff != "" {
		tb.Errorf("%s\nf.RunFunction(...): -want rsp, +got rsp:\n%s", c.Reason, Colorize(diff))
	}
	return expectFile
}

type readyMap map[string]fnv1.Ready

func (c *GoldenFileCase) verifyReady(tb gt.TB, currentReady readyMap, readyFileName string) {
	tb.Helper()
	cr, err := yaml.Marshal(currentReady)
	require.NoError(tb, err)
	if *update {
		err = os.WriteFile(readyFileName, cr, 0o600)
		require.NoError(tb, err)
		tb.Logf("Updated golden ready file %s", readyFileName)
	} else { // #nosec G304 -- expectFile is constructed from trusted test data (DesiredLocation and xrName)
		readyContent, err := os.ReadFile(readyFileName)
		require.NoError(tb, err)
		err = yaml.Unmarshal(readyContent, new(make(readyMap)))

		require.NoError(tb, err)
		diff, err := c.diff.unifiedYamlDiff(
			string(readyContent),
			string(cr),
			fmt.Sprintf("Ready (%s)", readyFileName),
			"Ready",
		)
		if err != nil {
			tb.Errorf("Error in diffing %s: %v", "Ready", err)
		} else if diff != "" {
			tb.Errorf(
				"%s\nf.RunFunction(...): -want rsp, +got rsp:\n%s",
				c.Reason,
				Colorize(diff),
			)
		}
	}
}

func sanitizeCaseName(caseName string) string {
	return invalidCaseNameCharRE.ReplaceAllString(caseName, "_")
}

func (c *GoldenFileCase) goldenFilePrefix() string {
	return fmt.Sprintf("%s-%s", c.caseName, c.xrName)
}

func (c *GoldenFileCase) goldenFileName(suffix string) string {
	return filepath.Join(c.DesiredLocation, fmt.Sprintf("%s-%s.yaml", c.goldenFilePrefix(), suffix))
}
