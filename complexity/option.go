package complexity

type Option func(*complexityOptions)

type complexityOptions struct {
	fixedScalarValue int
	ignoreFields     map[string]struct{}
}

var defaultOptions = complexityOptions{
	fixedScalarValue: 1,
	ignoreFields:     nil,
}

// WithFixedScalarValue sets the default value attributed to scalar and enum fields.
func WithFixedScalarValue(v int) Option {
	return func(o *complexityOptions) {
		o.fixedScalarValue = v
	}
}

// WithIgnoreFields specifies which fields are ignored in the complexity calculation.
// It's equivalent to setting the value of these fields to zero.
func WithIgnoreFields(m map[string]struct{}) Option {
	return func(o *complexityOptions) {
		o.ignoreFields = m
	}
}
