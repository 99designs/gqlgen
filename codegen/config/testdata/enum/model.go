package enum

type Bar int

const (
	BarOne Bar = iota + 1
	BarTwo
)

const (
	BazOne = iota + 1
	BazTwo
)
