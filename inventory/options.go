package inventory

// Option is a functional option for configuring BuildInventory.
type Option func(*builder) error

// WithComposite sets the composite (XRD) mapping function.
func WithComposite(m MappingFunc) Option {
	return func(opts *builder) error {
		opts.compositeFunc = m
		return nil
	}
}

// WithInput sets the input mapping function.
func WithInput(m MappingFunc) Option {
	return func(opts *builder) error {
		opts.inputFunc = m
		return nil
	}
}

// WithMapping sets the resource mappings.
func WithMapping(m Mapping) Option {
	return func(opts *builder) error {
		if err := m.validate(); err != nil {
			return err
		}
		opts.mapping = m
		return nil
	}
}
