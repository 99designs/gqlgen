package errors

import "fmt"

type GraphQLError struct {
	Message   string      `json:"message"`
	Locations []*Location `json:"locations,omitempty"`
}

type Location struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

func Errorf(format string, a ...interface{}) *GraphQLError {
	return &GraphQLError{
		Message: fmt.Sprintf(format, a...),
	}
}

func ErrorfWithLoc(line int, column int, format string, a ...interface{}) *GraphQLError {
	return &GraphQLError{
		Message: fmt.Sprintf(format, a...),
		Locations: []*Location{{
			Line:   line,
			Column: column,
		}},
	}
}

func (err *GraphQLError) Error() string {
	if err == nil {
		return "<nil>"
	}
	if len(err.Locations) > 0 {
		loc := err.Locations[0]
		return fmt.Sprintf("graphql: %s (line %d, column %d)", err.Message, loc.Line, loc.Column)
	}
	return fmt.Sprintf("graphql: %s", err.Message)
}

var _ error = &GraphQLError{}
