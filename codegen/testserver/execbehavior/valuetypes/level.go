package valuetypes

// Level is an int, bound to the GraphQL enum Level with enum_values.
type Level int

const (
	LevelLow Level = iota + 1
	LevelHigh
)
