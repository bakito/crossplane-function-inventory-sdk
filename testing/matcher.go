package testing

import (
	"errors"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	gt "testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// StringMatcher matches a string value, such as an error message or the message of a
// function result. Use Equal, Contains, Matching or MatchingRegexp to create one.
type StringMatcher interface {
	// MatchString reports whether got satisfies the matcher. Implementations apply normalize
	// to got, and to their own expectation wherever it is compared literally.
	MatchString(got string, normalize Normalize) bool
	// String describes the expectation. It is used in assertion failure output.
	String() string
}

type stringMatcher struct {
	desc  string
	match func(got string, normalize Normalize) bool
}

func (m *stringMatcher) MatchString(got string, normalize Normalize) bool {
	return m.match(got, normalize)
}

func (m *stringMatcher) String() string {
	return m.desc
}

// Equal matches a message that equals want once both sides are normalized.
func Equal(want string) StringMatcher {
	return &stringMatcher{
		desc: "message equal to " + strconv.Quote(want),
		match: func(got string, normalize Normalize) bool {
			return normalize(got) == normalize(want)
		},
	}
}

// Contains matches a message that contains all the given substrings. Prefer it over an exact
// match when part of the message comes from a library that randomizes its wording.
func Contains(substrs ...string) StringMatcher {
	quoted := make([]string, 0, len(substrs))
	for _, s := range substrs {
		quoted = append(quoted, strconv.Quote(s))
	}
	return &stringMatcher{
		desc: "message containing " + strings.Join(quoted, " and "),
		match: func(got string, normalize Normalize) bool {
			got = normalize(got)
			for _, want := range substrs {
				if !strings.Contains(got, normalize(want)) {
					return false
				}
			}
			return true
		},
	}
}

// Matching matches a message matching the given regular expression.
// It panics if pattern does not compile, just like regexp.MustCompile.
func Matching(pattern string) StringMatcher {
	return MatchingRegexp(regexp.MustCompile(pattern))
}

// MatchingRegexp matches a message matching the given regular expression.
func MatchingRegexp(re *regexp.Regexp) StringMatcher {
	return &stringMatcher{
		desc: "message matching " + strconv.Quote(re.String()),
		match: func(got string, normalize Normalize) bool {
			return re.MatchString(normalize(got))
		},
	}
}

// ErrorMatcher matches the error returned by the function under test. It implements error, so
// that it can be assigned to Case.Want.Err and GoldenFileCase.Err in place of a plain error.
type ErrorMatcher interface {
	error
	// MatchError reports whether got satisfies the matcher. normalize canonicalizes the
	// renderings that upstream libraries deliberately randomize.
	MatchError(got error, normalize Normalize) bool
}

// errMatcher implements ErrorMatcher. It is not an error itself: Error only describes the
// expectation, so that a matcher fits into the error-typed Err fields of the test cases.
//
//nolint:errname // not an error type, only error-shaped to fit the Err fields
type errMatcher struct {
	desc  string
	match func(got error, normalize Normalize) bool
}

func (m *errMatcher) Error() string {
	return m.desc
}

func (m *errMatcher) MatchError(got error, normalize Normalize) bool {
	return m.match(got, normalize)
}

// ErrorMessage matches an error whose message satisfies m.
func ErrorMessage(m StringMatcher) ErrorMatcher {
	return &errMatcher{
		desc: m.String(),
		match: func(got error, normalize Normalize) bool {
			return got != nil && m.MatchString(got.Error(), normalize)
		},
	}
}

// ErrorEqual matches an error whose message equals msg once both sides are normalized. It is
// the explicit form of assigning a plain error to the Err field of a test case.
func ErrorEqual(msg string) ErrorMatcher {
	return ErrorMessage(Equal(msg))
}

// ErrorContains matches an error whose message contains all the given substrings. This is the
// simplest way to pin the stable part of a wrapped error while ignoring library wording that
// changes between runs.
func ErrorContains(substrs ...string) ErrorMatcher {
	return ErrorMessage(Contains(substrs...))
}

// ErrorMatching matches an error whose message matches the given regular expression.
// It panics if pattern does not compile, just like regexp.MustCompile.
func ErrorMatching(pattern string) ErrorMatcher {
	return ErrorMessage(Matching(pattern))
}

// ErrorIs matches an error that is target according to errors.Is. This is the most robust
// check available, as it does not look at the message at all.
func ErrorIs(target error) ErrorMatcher {
	desc := "<nil>"
	if target != nil {
		desc = strconv.Quote(target.Error())
	}
	return &errMatcher{
		desc: "error is " + desc,
		match: func(got error, _ Normalize) bool {
			return errors.Is(got, target)
		},
	}
}

// ErrorAs matches an error that wraps an error of type T according to errors.As. Use it to
// assert the kind of a library error, for example ErrorAs[*json.SemanticError]().
func ErrorAs[T error]() ErrorMatcher {
	return &errMatcher{
		desc: "error as " + reflect.TypeFor[T]().String(),
		match: func(got error, _ Normalize) bool {
			var target T
			return errors.As(got, &target)
		},
	}
}

// AllOf matches an error that satisfies all the given matchers. Combine a kind with its
// context, for example AllOf(ErrorAs[*json.SemanticError](), ErrorContains("cannot marshal")).
func AllOf(matchers ...ErrorMatcher) ErrorMatcher {
	descs := make([]string, 0, len(matchers))
	for _, m := range matchers {
		descs = append(descs, m.Error())
	}
	return &errMatcher{
		desc: strings.Join(descs, " and "),
		match: func(got error, normalize Normalize) bool {
			for _, m := range matchers {
				if !m.MatchError(got, normalize) {
					return false
				}
			}
			return true
		},
	}
}

// verifyError asserts that the error returned by the function under test is the expected one.
// A want of nil requires no error. A want implementing ErrorMatcher is applied as such, any
// other error matches either by errors.Is or by its normalized message.
func verifyError(tb gt.TB, d *diff, got, want error) {
	tb.Helper()
	if want == nil {
		require.NoError(tb, got)
		return
	}
	require.Error(tb, got)
	if got == nil {
		// Only reached with a gt.TB whose FailNow does not abort, as in the package tests.
		return
	}
	var m ErrorMatcher
	if errors.As(want, &m) {
		if !m.MatchError(got, d.normalize) {
			require.Failf(tb, "unexpected error", "want: %s\ngot:  %s", m, got)
		}
		return
	}
	if errors.Is(got, want) {
		return
	}
	require.Equal(tb, d.normalize(want.Error()), d.normalize(got.Error()), "unexpected error message")
}

// verifyMessage asserts a function result message against want, or against matcher when set.
func verifyMessage(tb gt.TB, d *diff, got, want string, matcher StringMatcher) {
	tb.Helper()
	switch {
	case matcher != nil:
		if !matcher.MatchString(got, d.normalize) {
			// assert, so that the remaining results are still verified
			assert.Failf(tb, "unexpected result message", "want: %s\ngot:  %s", matcher, got)
		}
	case want != "":
		assert.Equal(tb, d.normalize(want), d.normalize(got), "unexpected result message")
	}
}
