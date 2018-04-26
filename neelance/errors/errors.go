package errors

import (
	"fmt"
)

type QueryError struct {
	Message       string        `json:"message"`
	Locations     []Location    `json:"locations,omitempty"`
	Path          []interface{} `json:"path,omitempty"`
	Rule          string        `json:"-"`
	ResolverError error         `json:"-"`
	ExtraInfo     interface{}   `json:"extraInfo,omitempty"`
}

type Location struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

func (a Location) Before(b Location) bool {
	return a.Line < b.Line || (a.Line == b.Line && a.Column < b.Column)
}

func Errorf(format string, a ...interface{}) *QueryError {
	return &QueryError{
		Message: fmt.Sprintf(format, a...),
	}
}

// WithMessagef is the same as Errorf, except it will store the err inside
// the ResolverError field.
func WithMessagef(err error, extra interface{}, format string, a ...interface{}) *QueryError {
	return &QueryError{
		Message:       fmt.Sprintf(format, a...),
		ResolverError: err,
		ExtraInfo:     extra,
	}
}

func (err *QueryError) Error() string {
	if err == nil {
		return "<nil>"
	}
	str := fmt.Sprintf("graphql: %s", err.Message)
	for _, loc := range err.Locations {
		str += fmt.Sprintf(" (line %d, column %d)", loc.Line, loc.Column)
	}
	return str
}

var _ error = &QueryError{}

// FormattedError a formatted error which includes the error message and extra information
// which is JSON encoded and passed with the error.
type FormattedError struct {
	Message string
	Extra   interface{}
}

// ErrorMessageFunc a func which given an error returns a formatted error.
type ErrorMessageFunc func(err error) FormattedError

type Builder struct {
	Errors []*QueryError
	// ErrorMessageFn will be used to generate the error
	// message from errors given to Error().
	//
	// If ErrorMessageFn is nil, err.Error() will be used.
	ErrorMessageFn ErrorMessageFunc
}

func (c *Builder) Errorf(format string, args ...interface{}) {
	c.Errors = append(c.Errors, Errorf(format, args...))
}

func (c *Builder) Error(err error) {
	fErr := FormattedError{Message: err.Error()}

	if c.ErrorMessageFn != nil {
		fErr = c.ErrorMessageFn(err)
	}

	c.Errors = append(c.Errors, WithMessagef(err, fErr.Extra, fErr.Message))
}
