package errors

import (
	"fmt"
	"io"

	"github.com/vektah/gqlgen/jsonw"
)

type QueryError struct {
	Message       string        `json:"message"`
	Locations     []Location    `json:"locations,omitempty"`
	Path          []interface{} `json:"path,omitempty"`
	Rule          string        `json:"-"`
	ResolverError error         `json:"-"`
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

type Builder struct {
	Errors []*QueryError
}

func (c *Builder) Errorf(format string, args ...interface{}) {
	c.Errors = append(c.Errors, Errorf(format, args...))
}

func (c *Builder) Error(err error) {
	c.Errors = append(c.Errors, Errorf("%s", err.Error()))
}

func WriteErrors(b io.Writer, errs []*QueryError) {
	w := jsonw.New(b)
	w.BeginArray()
	for _, err := range errs {
		if err == nil {
			w.Null()
			continue
		}
		w.BeginObject()

		w.ObjectKey("message")
		w.String(err.Message)

		if len(err.Locations) > 0 {
			w.ObjectKey("locations")
			w.BeginArray()
			for _, location := range err.Locations {
				w.BeginObject()

				w.ObjectKey("line")
				w.Int(location.Line)

				w.ObjectKey("column")
				w.Int(location.Column)

				w.EndObject()
			}
			w.EndArray()
		}

		w.EndObject()
	}
	w.EndArray()
}
