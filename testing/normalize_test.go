package testing

import (
	"regexp"
	"strings"
	gt "testing"

	"github.com/go-json-experiment/json"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

// The two renderings github.com/go-json-experiment/json picks between, once per process.
// Named without "json", so that testifylint does not mistake them for JSON documents.
const (
	msgVerbCannot   = `cannot marshal to JSON: json: cannot unmarshal JSON number into Go string`
	msgVerbUnableTo = `cannot marshal to JSON: json: unable to unmarshal JSON number into Go string`
)

func TestNormalizeErrorMessage(t *gt.T) {
	tests := map[string]struct {
		in   string
		want string
	}{
		"json modal verb unable to": {
			in:   msgVerbUnableTo,
			want: msgVerbCannot,
		},
		"json modal verb cannot is left alone": {
			in:   msgVerbCannot,
			want: msgVerbCannot,
		},
		"proto non-breaking space": {
			in:   "proto:\u00a0cannot parse invalid wire-format data",
			want: "proto: cannot parse invalid wire-format data",
		},
		"proto regular space is left alone": {
			in:   "proto: cannot parse invalid wire-format data",
			want: "proto: cannot parse invalid wire-format data",
		},
		"both at once": {
			in:   "json: unable to unmarshal: proto:\u00a0bad",
			want: "json: cannot unmarshal: proto: bad",
		},
		"unable to outside of a json error is left alone": {
			in:   "the controller was unable to reconcile",
			want: "the controller was unable to reconcile",
		},
		"non-breaking space in resource data is left alone": {
			in:   "name: some\u00a0value",
			want: "name: some\u00a0value",
		},
		"empty": {
			in:   "",
			want: "",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *gt.T) {
			got := NormalizeErrorMessage(tc.in)
			assert.Equal(t, tc.want, got)
			assert.Equal(t, got, NormalizeErrorMessage(got), "must be idempotent")
		})
	}
}

// TestNormalizeErrorMessage_realErrors pins the normalized rendering of errors that actually
// come out of the libraries. Without normalization these assertions fail in roughly one of
// twelve processes for the json error, and on every build whose protobuf prefix is drawn with
// non-breaking spaces for the proto error.
func TestNormalizeErrorMessage_realErrors(t *gt.T) {
	t.Run("wrapped json error", func(t *gt.T) {
		var target struct{ A string }
		err := json.Unmarshal([]byte(`{"A":1}`), &target)
		require.Error(t, err)
		wrapped := errors.Wrap(err, "cannot marshal to JSON")
		// Logged so that a future flake here shows which rendering the process picked.
		t.Logf("this process renders json errors as %q", err.Error())

		assert.Equal(t,
			`cannot marshal to JSON: json: cannot unmarshal JSON number into Go string within "/A"`,
			NormalizeErrorMessage(wrapped.Error()),
		)
	})

	t.Run("protojson error", func(t *gt.T) {
		err := protojson.Unmarshal([]byte(`{"x":`), &structpb.Struct{})
		require.Error(t, err)

		got := NormalizeErrorMessage(err.Error())
		assert.True(t, strings.HasPrefix(got, "proto: "), "want a regular space prefix, got %q", got)
		assert.NotContains(t, got, "\u00a0")
	})
}

func TestDiffNormalize(t *gt.T) {
	t.Run("defaults are applied", func(t *gt.T) {
		d := newDiff(nil)
		assert.Equal(t, msgVerbCannot, d.normalize(msgVerbUnableTo))
	})

	t.Run("custom normalizer is applied on top of the defaults", func(t *gt.T) {
		d := newDiff([]DiffOption{
			WithNormalizer(regexp.MustCompile(`after offset \d+`), "after offset N"),
		})
		assert.Equal(t,
			"json: cannot unmarshal after offset N",
			d.normalize("json: unable to unmarshal after offset 123"),
		)
	})

	t.Run("defaults can be disabled", func(t *gt.T) {
		d := newDiff([]DiffOption{WithoutDefaultNormalizers()})
		assert.Equal(t, msgVerbUnableTo, d.normalize(msgVerbUnableTo))
	})

	t.Run("disabling the defaults keeps custom normalizers", func(t *gt.T) {
		d := newDiff([]DiffOption{
			WithNormalizer(regexp.MustCompile(`\d+`), "N"),
			WithoutDefaultNormalizers(),
		})
		assert.Equal(t, "json: unable to unmarshal after offset N",
			d.normalize("json: unable to unmarshal after offset 123"))
	})
}
