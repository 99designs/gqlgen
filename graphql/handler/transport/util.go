package transport

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/gqlerror"
)

func writeJson(w io.Writer, response *graphql.Response) {
	b, err := json.Marshal(response)
	if err != nil {
		panic(err)
	}
	w.Write(b)
}

func writeJsonError(w io.Writer, msg string) {
	writeJson(w, &graphql.Response{Errors: gqlerror.List{{Message: msg}}})
}

func writeJsonErrorf(w io.Writer, format string, args ...interface{}) {
	writeJson(w, &graphql.Response{Errors: gqlerror.List{{Message: fmt.Sprintf(format, args...)}}})
}

func writeJsonGraphqlError(w io.Writer, err ...*gqlerror.Error) {
	writeJson(w, &graphql.Response{Errors: err})
}

func httpStatusCode(resp *graphql.Response) int {
	if len(resp.Errors) == 0 {
		return http.StatusOK
	}

	if len(resp.Data) != 0 {
		return http.StatusOK
	} else if len(resp.Errors) == 1 && resp.Errors[0].Message == "PersistedQueryNotFound" {
		// for APQ
		return http.StatusOK
	}

	return http.StatusUnprocessableEntity
}
