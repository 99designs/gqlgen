package errors

import "fmt"

type QueryError struct {
	Message       string     `json:"message"`
	Locations     []Location `json:"locations,omitempty"`
	Rule          string     `json:"-"`
	ResolverError error      `json:"-"`
}

type Location struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

func Errorf(format string, a ...interface{}) *QueryError {
	return &QueryError{
		Message: fmt.Sprintf(format, a...),
	}
}

func (err *QueryError) Error() string {
	if err == nil {
		return "<nil>"
	}
	if len(err.Locations) > 0 {
		loc := err.Locations[0]
		return fmt.Sprintf("graphql: %s (line %d, column %d)", err.Message, loc.Line, loc.Column)
	}
	return fmt.Sprintf("graphql: %s", err.Message)
}

var _ error = &QueryError{}
