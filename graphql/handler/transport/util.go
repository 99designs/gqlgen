package transport

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/vektah/gqlparser/v2/gqlerror"

	"github.com/99designs/gqlgen/graphql"
)

type JsonWriter interface {
	WriteJson(*graphql.Response)
}

func writeJson(w http.ResponseWriter, response *graphql.Response) {
	if jsonWriter, ok := w.(JsonWriter); ok {
		jsonWriter.WriteJson(response)
		return
	}

	b, err := json.Marshal(response)
	if err != nil {
		panic(fmt.Errorf("unable to marshal %s: %w", string(response.Data), err))
	}
	_, _ = w.Write(b)
}

func writeJsonError(w http.ResponseWriter, msg string) {
	writeJson(w, &graphql.Response{Errors: gqlerror.List{{Message: msg}}})
}

func writeJsonErrorf(w http.ResponseWriter, format string, args ...any) {
	writeJson(w, &graphql.Response{Errors: gqlerror.List{{Message: fmt.Sprintf(format, args...)}}})
}

func writeJsonGraphqlError(w http.ResponseWriter, err ...*gqlerror.Error) {
	writeJson(w, &graphql.Response{Errors: err})
}
