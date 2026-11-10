package testing

import (
	gt "testing"

	"github.com/go-json-experiment/json"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errSentinel = errors.New("sentinel")

func TestStringMatchers(t *gt.T) {
	normalize := newDiff(nil).normalize

	tests := map[string]struct {
		matcher  StringMatcher
		got      string
		want     bool
		wantDesc string
	}{
		"equal matches": {
			matcher:  Equal("boom"),
			got:      "boom",
			want:     true,
			wantDesc: `message equal to "boom"`,
		},
		"equal does not match a different message": {
			matcher: Equal("boom"),
			got:     "bang",
		},
		"equal normalizes both sides": {
			matcher: Equal(msgVerbUnableTo),
			got:     msgVerbCannot,
			want:    true,
		},
		"contains matches a substring": {
			matcher:  Contains("cannot marshal"),
			got:      msgVerbCannot,
			want:     true,
			wantDesc: `message containing "cannot marshal"`,
		},
		"contains requires every substring": {
			matcher: Contains("cannot marshal", "nope"),
			got:     msgVerbCannot,
		},
		"contains matches every substring": {
			matcher:  Contains("cannot marshal", "JSON number"),
			got:      msgVerbCannot,
			want:     true,
			wantDesc: `message containing "cannot marshal" and "JSON number"`,
		},
		"contains normalizes the expectation too": {
			matcher: Contains("json: unable to unmarshal"),
			got:     msgVerbCannot,
			want:    true,
		},
		"matching matches a pattern": {
			matcher:  Matching(`^cannot marshal .* JSON`),
			got:      msgVerbCannot,
			want:     true,
			wantDesc: `message matching "^cannot marshal .* JSON"`,
		},
		"matching does not match a different message": {
			matcher: Matching(`^nope`),
			got:     msgVerbCannot,
		},
		"matching normalizes the message": {
			matcher: Matching(`json: cannot unmarshal`),
			got:     msgVerbUnableTo,
			want:    true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *gt.T) {
			assert.Equal(t, tc.want, tc.matcher.MatchString(tc.got, normalize))
			if tc.wantDesc != "" {
				assert.Equal(t, tc.wantDesc, tc.matcher.String())
			}
		})
	}
}

func TestErrorMatchers(t *gt.T) {
	normalize := newDiff(nil).normalize

	var jsonTarget struct{ A string }
	jsonErr := json.Unmarshal([]byte(`{"A":1}`), &jsonTarget)
	require.Error(t, jsonErr)
	wrappedJSON := errors.Wrap(jsonErr, "cannot marshal to JSON")
	wrappedSentinel := errors.Wrap(errSentinel, "while reading the input")

	tests := map[string]struct {
		matcher  ErrorMatcher
		got      error
		want     bool
		wantDesc string
	}{
		"equal matches regardless of the modal verb": {
			matcher: ErrorEqual(msgVerbUnableTo),
			got:     errors.New(msgVerbCannot),
			want:    true,
		},
		"contains pins the stable part of a wrapped library error": {
			matcher:  ErrorContains("cannot marshal to JSON", "JSON number into Go string"),
			got:      wrappedJSON,
			want:     true,
			wantDesc: `message containing "cannot marshal to JSON" and "JSON number into Go string"`,
		},
		"contains does not match a different error": {
			matcher: ErrorContains("cannot marshal to JSON"),
			got:     errors.New("something else entirely"),
		},
		"matching matches a pattern": {
			matcher: ErrorMatching(`^cannot marshal to JSON: json: cannot unmarshal`),
			got:     wrappedJSON,
			want:    true,
		},
		"is matches a wrapped sentinel": {
			matcher:  ErrorIs(errSentinel),
			got:      wrappedSentinel,
			want:     true,
			wantDesc: `error is "sentinel"`,
		},
		"is does not match an unrelated error": {
			matcher: ErrorIs(errSentinel),
			got:     errors.New("sentinel"),
		},
		"as matches a wrapped library error type": {
			matcher:  ErrorAs[*json.SemanticError](),
			got:      wrappedJSON,
			want:     true,
			wantDesc: "error as *json.SemanticError",
		},
		"as does not match another type": {
			matcher: ErrorAs[*json.SemanticError](),
			got:     wrappedSentinel,
		},
		"all of requires every matcher": {
			matcher: AllOf(ErrorAs[*json.SemanticError](), ErrorContains("nope")),
			got:     wrappedJSON,
		},
		"all of matches every matcher": {
			matcher:  AllOf(ErrorAs[*json.SemanticError](), ErrorContains("cannot marshal to JSON")),
			got:      wrappedJSON,
			want:     true,
			wantDesc: `error as *json.SemanticError and message containing "cannot marshal to JSON"`,
		},
		"message bridges any string matcher": {
			matcher: ErrorMessage(Contains("cannot marshal to JSON")),
			got:     wrappedJSON,
			want:    true,
		},
		"a nil error never matches": {
			matcher: ErrorContains("cannot marshal to JSON"),
			got:     nil,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *gt.T) {
			assert.Equal(t, tc.want, tc.matcher.MatchError(tc.got, normalize))
			if tc.wantDesc != "" {
				assert.Equal(t, tc.wantDesc, tc.matcher.Error())
			}
		})
	}
}

func TestErrorIsNilDescription(t *gt.T) {
	assert.Equal(t, "error is <nil>", ErrorIs(nil).Error())
}

func TestVerifyError(t *gt.T) {
	d := newDiff(nil)

	tests := map[string]struct {
		got        error
		want       error
		shouldFail bool
	}{
		"no error wanted, none returned": {},
		"no error wanted, one returned": {
			got:        errors.New("boom"),
			shouldFail: true,
		},
		"error wanted, none returned": {
			want:       errors.New("boom"),
			shouldFail: true,
		},
		"matching message": {
			got:  errors.New("boom"),
			want: errors.New("boom"),
		},
		"different message": {
			got:        errors.New("bang"),
			want:       errors.New("boom"),
			shouldFail: true,
		},
		"message differing only in the modal verb": {
			got:  errors.New(msgVerbCannot),
			want: errors.New(msgVerbUnableTo),
		},
		"wrapped sentinel matches by errors.Is": {
			got:  errors.Wrap(errSentinel, "while reading the input"),
			want: errSentinel,
		},
		"matcher is satisfied": {
			got:  errors.New(msgVerbCannot),
			want: ErrorContains("cannot marshal to JSON"),
		},
		"matcher is not satisfied": {
			got:        errors.New(msgVerbCannot),
			want:       ErrorContains("nope"),
			shouldFail: true,
		},
		"matcher wanted, no error returned": {
			want:       ErrorContains("cannot marshal to JSON"),
			shouldFail: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *gt.T) {
			mt := &mockTesting{T: t, shouldFail: tc.shouldFail}
			verifyError(mt, d, tc.got, tc.want)
			assert.Equal(t, tc.shouldFail, mt.failed)
		})
	}
}
