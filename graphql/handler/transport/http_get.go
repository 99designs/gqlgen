package transport

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/99designs/gqlgen/graphql"
)

type HTTPGet struct{}

var _ graphql.Transport = HTTPGet{}

func (H HTTPGet) Supports(r *http.Request) bool {
	if r.Header.Get("Upgrade") != "" {
		return false
	}

	return r.Method == "GET"
}

func (H HTTPGet) Do(w http.ResponseWriter, r *http.Request, handler graphql.Handler) {
	rc := newRequestContext()
	rc.RawQuery = r.URL.Query().Get("query")
	rc.OperationName = r.URL.Query().Get("operationName")

	writer := graphql.Writer(func(response *graphql.Response) {
		b, err := json.Marshal(response)
		if err != nil {
			panic(err)
		}
		w.Write(b)
	})

	if variables := r.URL.Query().Get("variables"); variables != "" {
		if err := jsonDecode(strings.NewReader(variables), &rc.Variables); err != nil {
			writer.Errorf("variables could not be decoded")
			return
		}
	}

	if extensions := r.URL.Query().Get("extensions"); extensions != "" {
		if err := jsonDecode(strings.NewReader(extensions), &rc.Extensions); err != nil {
			writer.Errorf("extensions could not be decoded")
			return
		}
	}

	// TODO: FIXME
	//if op.Operation != ast.Query && args.R.Method == http.MethodGet {
	//	return ctx, nil, nil, gqlerror.List{gqlerror.Errorf("GET requests only allow query operations")}
	//}

	ctx := graphql.WithRequestContext(r.Context(), rc)
	handler(ctx, writer)
}

func jsonDecode(r io.Reader, val interface{}) error {
	dec := json.NewDecoder(r)
	dec.UseNumber()
	return dec.Decode(val)
}
