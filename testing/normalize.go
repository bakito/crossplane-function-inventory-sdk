package testing

import (
	"regexp"
	"strings"
)

const (
	// jsonErrorPrefix is the prefix of every error returned by github.com/go-json-experiment/json.
	jsonErrorPrefix = "json: "
	// protoErrorPrefix is the prefix of every error returned by google.golang.org/protobuf.
	protoErrorPrefix = "proto:"
	// nbsp is the non-breaking space (U+00A0) protobuf may use instead of a regular space.
	nbsp = "\u00a0"
)

// Normalize canonicalizes unstable renderings in a message before it is compared.
type Normalize func(s string) string

// Normalizer replaces a deliberately unstable fragment of a message.
// Use NewNormalizer to create one and WithNormalizer to register it on a test case.
type Normalizer struct {
	re   *regexp.Regexp
	repl string
}

// NewNormalizer returns a Normalizer that replaces every match of re with repl.
func NewNormalizer(re *regexp.Regexp, repl string) Normalizer {
	return Normalizer{re: re, repl: repl}
}

func (n Normalizer) apply(s string) string {
	if n.re == nil {
		return s
	}
	return n.re.ReplaceAllString(s, n.repl)
}

// NormalizeErrorMessage canonicalizes the renderings that upstream libraries deliberately
// randomize in order to discourage comparing error strings:
//
//   - github.com/go-json-experiment/json picks the modal verb "cannot" or "unable to" once
//     per process (errorModalVerb in its errors.go), so "json: unable to" is rewritten to
//     "json: cannot".
//   - google.golang.org/protobuf prefixes its errors with "proto: " using either a regular
//     space (U+0020) or a non-breaking space (U+00A0), chosen from a hash of the test binary
//     (prefixError in its internal/errors), so the non-breaking variant is rewritten to the
//     regular one.
//
// Both substitutions are anchored on the library prefix, so arbitrary resource text is never
// touched. The function is idempotent.
//
// It is applied to both sides of every comparison made by Case and GoldenFileCase. Use it
// directly in hand-written tests that compare such messages themselves.
func NormalizeErrorMessage(s string) string {
	s = strings.ReplaceAll(s, jsonErrorPrefix+"unable to ", jsonErrorPrefix+"cannot ")
	return strings.ReplaceAll(s, protoErrorPrefix+nbsp, protoErrorPrefix+" ")
}
