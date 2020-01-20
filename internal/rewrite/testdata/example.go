package testdata

type Foo struct {
	Field int
}

func (m *Foo) Method(arg int) {
	// leading comment

	// field comment
	m.Field++

	// trailing comment
}
