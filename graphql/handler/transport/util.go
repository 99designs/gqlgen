package transport

import (
	"fmt"
	"io"

	"github.com/99designs/gqlgen/graphql/handler/serial"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

func writeJson(w io.Writer, serial serial.Serialization, response *graphql.Response) {
	b, err := serial.Marshal(response)
	if err != nil {
		panic(err)
	}
	w.Write(b)
}

func writeJsonError(w io.Writer, serial serial.Serialization, msg string) {
	writeJson(w, serial, &graphql.Response{Errors: gqlerror.List{{Message: msg}}})
}

func writeJsonErrorf(w io.Writer, serial serial.Serialization, format string, args ...interface{}) {
	writeJson(w, serial, &graphql.Response{Errors: gqlerror.List{{Message: fmt.Sprintf(format, args...)}}})
}

func writeJsonGraphqlError(w io.Writer, serial serial.Serialization, err ...*gqlerror.Error) {
	writeJson(w, serial, &graphql.Response{Errors: err})
}
