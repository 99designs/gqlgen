package graphql

// Omittable is a wrapper around a value that also stores whether it is set
// or not.
type Omittable[T any] struct {
	value T
	set   bool
}

func OmittableOf[T any](value T) Omittable[T] {
	return Omittable[T]{
		value: value,
		set:   true,
	}
}

func (o Omittable[T]) Value() T {
	if !o.set {
		var zero T
		return zero
	}
	return o.value
}

func (o Omittable[T]) ValueOK() (T, bool) {
	if !o.set {
		var zero T
		return zero, false
	}
	return o.value, true
}

func (o Omittable[T]) IsSet() bool {
	return o.set
}
