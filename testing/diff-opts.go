package testing

import "regexp"

type DiffOption func(opts *diff)

// WithDiffContextLines sets the number of context lines to use when diffing YAML.
func WithDiffContextLines(lines int) DiffOption {
	return func(opts *diff) {
		opts.diffContextLines = lines
	}
}

// WithYamlIndent sets the YAML indentation level.
func WithYamlIndent(indent int) DiffOption {
	return func(opts *diff) {
		opts.yamlIndent = indent
	}
}

// WithNormalizer registers an additional replacement applied to both sides of every
// comparison: error messages, result messages, response diffs and golden files. Use it for
// values that change between runs, such as byte offsets, timestamps or UUIDs.
func WithNormalizer(re *regexp.Regexp, repl string) DiffOption {
	return func(opts *diff) {
		opts.normalizers = append(opts.normalizers, NewNormalizer(re, repl))
	}
}

// WithoutDefaultNormalizers disables the built-in normalizers described in
// NormalizeErrorMessage, making every comparison byte-exact again.
func WithoutDefaultNormalizers() DiffOption {
	return func(opts *diff) {
		opts.noDefaultNormalizers = true
	}
}

// Granular if enabled, diff is evaluated on individual response parts as yaml, rather dnt on the whole structpb response object.
func Granular() DiffOption {
	return func(opts *diff) {
		opts.granular = true
	}
}
