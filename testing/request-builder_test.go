package testing

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
)

const testResourceFile = "testdata/resource.yaml"

func TestRunFunctionRequestBuilder_WithDesiredComposite(t *testing.T) {
	tests := []struct {
		name  string
		build func(*RunFunctionRequestBuilder) *RunFunctionRequestBuilder
	}{
		{
			name: "WithDesiredComposite",
			build: func(b *RunFunctionRequestBuilder) *RunFunctionRequestBuilder {
				return b.WithDesiredComposite(From(testResourceFile))
			},
		},
		{
			name: "Wdc alias",
			build: func(b *RunFunctionRequestBuilder) *RunFunctionRequestBuilder {
				return b.Wdc(From(testResourceFile))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.build(NewRequestBuilder()).Build()
			require.NotNil(t, req.GetDesired().GetComposite())
			assert.Equal(
				t,
				"test-resource",
				req.GetDesired().GetComposite().GetResource().GetFields()["metadata"].
					GetStructValue().GetFields()["name"].GetStringValue(),
			)
		})
	}
}

func TestRunFunctionRequestBuilder_WithDesiredResource(t *testing.T) {
	tests := []struct {
		name      string
		build     func(*RunFunctionRequestBuilder) *RunFunctionRequestBuilder
		wantReady fnv1.Ready
		wantPanic bool
	}{
		{
			name: "without ready",
			build: func(b *RunFunctionRequestBuilder) *RunFunctionRequestBuilder {
				return b.WithDesiredResource("res-a", From(testResourceFile))
			},
			wantReady: fnv1.Ready_READY_UNSPECIFIED,
		},
		{
			name: "with ready",
			build: func(b *RunFunctionRequestBuilder) *RunFunctionRequestBuilder {
				return b.WithDesiredResource("res-a", From(testResourceFile), fnv1.Ready_READY_TRUE)
			},
			wantReady: fnv1.Ready_READY_TRUE,
		},
		{
			name: "Wdr alias",
			build: func(b *RunFunctionRequestBuilder) *RunFunctionRequestBuilder {
				return b.Wdr("res-a", From(testResourceFile))
			},
			wantReady: fnv1.Ready_READY_UNSPECIFIED,
		},
		{
			name: "duplicate name panics",
			build: func(b *RunFunctionRequestBuilder) *RunFunctionRequestBuilder {
				return b.
					WithDesiredResource("res-a", From(testResourceFile)).
					WithDesiredResource("res-a", From(testResourceFile))
			},
			wantPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantPanic {
				assert.Panics(t, func() { tt.build(NewRequestBuilder()) })
				return
			}

			req := tt.build(NewRequestBuilder()).Build()
			require.Contains(t, req.GetDesired().GetResources(), "res-a")
			assert.Equal(t, tt.wantReady, req.GetDesired().GetResources()["res-a"].GetReady())
		})
	}
}
