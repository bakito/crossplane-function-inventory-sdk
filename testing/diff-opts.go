package testing

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

// Granular if enabled, diff is evaluated on individual response parts as yaml, rather dnt on the whole structpb response object.
func Granular() DiffOption {
	return func(opts *diff) {
		opts.granular = true
	}
}
