package testing

import (
	"bytes"
	"strings"
	gt "testing"

	"github.com/aymanbagabas/go-udiff"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const (
	red   = "\033[31m"
	green = "\033[32m"
	reset = "\033[0m"

	diffContextLines = 5
	yamlIndent       = 2
)

type diff struct {
	diffContextLines int
	yamlIndent       int
	granular         bool
}

// Colorize returns a string with colorized diff.
func Colorize(diff string) string {
	var colored strings.Builder
	for line := range strings.SplitSeq(diff, "\n") {
		switch {
		case strings.HasPrefix(line, "-"):
			colored.WriteString(red + line + reset + "\n")
		case strings.HasPrefix(line, "+"):
			colored.WriteString(green + line + reset + "\n")
		default:
			colored.WriteString(line + "\n")
		}
	}
	return colored.String()
}

func (d *diff) unifiedDiff(want, actual any, description string) (string, error) {
	yamlWant, err := d.toYaml(want)
	if err != nil {
		return "", err
	}
	yamlActual, err := d.toYaml(actual)
	if err != nil {
		return "", err
	}

	return d.unifiedYamlDiff(yamlWant, yamlActual, description, description)
}

func (d *diff) unifiedYamlDiff(
	yamlExpected, yamlActual, descExpected, descActual string,
) (string, error) {
	edits := udiff.Strings(yamlExpected, yamlActual)
	return udiff.ToUnified(
		"Expected "+descExpected,
		"Actual "+descActual,
		yamlExpected,
		edits,
		d.diffContextLines,
	)
}

func (d *diff) toYaml(a any) (string, error) {
	buf := bytes.Buffer{}
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(d.yamlIndent)
	// Can set default indent here on the encoder
	if err := enc.Encode(a); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// AssertEqual asserts that two values are equal. Values are converted to YAML before comparison, differences are represented as colored diff.
func AssertEqual(t *gt.T, want, actual any, description string, opts ...DiffOption) {
	t.Helper()
	d := newDiff(opts)

	diff, err := d.unifiedDiff(want, actual, description)
	require.NoError(t, err)
	if diff != "" {
		t.Errorf("%s do not match\n%s", description, Colorize(diff))
	}
}

func newDiff(opts []DiffOption) *diff {
	d := &diff{
		diffContextLines: diffContextLines,
		yamlIndent:       yamlIndent,
	}

	for _, opt := range opts {
		opt(d)
	}
	return d
}
