package main

import (
	"os"
	gt "testing"

	"github.com/bakito/crossplane-function-inventory-sdk/testing"
	"github.com/stretchr/testify/require"
)

func TestConstGenerator(t *gt.T) {
	tests := []struct {
		name       string
		inputFile  string
		expectFile string
	}{
		{
			name:       "input file with simple fields",
			inputFile:  "../../testdata/tag-const-gen/test-simple-struct.go",
			expectFile: "../../testdata/tag-const-gen/test-simple-struct.expected",
		},
		{
			name:       "input file with map fields",
			inputFile:  "../../testdata/tag-const-gen/test-map-struct.go",
			expectFile: "../../testdata/tag-const-gen/test-map-struct.expected",
		},
		{
			name:       "input file with two types and redundant prefix",
			inputFile:  "../../testdata/tag-const-gen/test-multiple-structs-name-dedup.go",
			expectFile: "../../testdata/tag-const-gen/test-multiple-structs-name-dedup.expected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *gt.T) {
			got := newConstGenerator(tt.inputFile)
			err := got.generateConstants()
			require.NoError(t, err)

			gotContent, err := os.ReadFile(got.outputFile)
			require.NoError(t, err, "failed to read generated file")

			wantContent, err := os.ReadFile(tt.expectFile)
			require.NoError(t, err, "failed to read expected file")

			testing.AssertEqual(t, string(wantContent), string(gotContent), "output file")
		})
	}
}
