package transport

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler/cache"
	"github.com/vektah/gqlparser/v2/gqlerror"
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

func writeCacheControl(w http.ResponseWriter, response *graphql.Response) {
	if len(response.Errors) > 0 {
		return
	}

	if cachePolicy, ok := cache.GetOverallCachePolicy(response); ok {
		cacheControl := fmt.Sprintf("max-age: %v %s", cachePolicy.MaxAge, strings.ToLower(string(cachePolicy.Scope)))
		w.Header().Add("Cache-Control", cacheControl)
	}
}
