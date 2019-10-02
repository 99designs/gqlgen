package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/99designs/gqlgen/graphql"
)

type HTTPGet struct{}

func (H HTTPGet) Supports(r *http.Request) bool {
	if r.Header.Get("Upgrade") != "" {
		return false
	}

	return r.Method == "GET"
}

func (H HTTPGet) Do(w http.ResponseWriter, r *http.Request) (*graphql.RequestContext, Writer) {
	reqParams := newRequestContext()
	reqParams.RawQuery = r.URL.Query().Get("query")
	reqParams.OperationName = r.URL.Query().Get("operationName")

	if variables := r.URL.Query().Get("variables"); variables != "" {
		if err := jsonDecode(strings.NewReader(variables), &reqParams.Variables); err != nil {
			sendErrorf(w, http.StatusBadRequest, "variables could not be decoded")
			return nil, nil
		}
	}

	if extensions := r.URL.Query().Get("extensions"); extensions != "" {
		if err := jsonDecode(strings.NewReader(extensions), &reqParams.Extensions); err != nil {
			sendErrorf(w, http.StatusBadRequest, "extensions could not be decoded")
			return nil, nil
		}
	}

	// TODO: FIXME
	//if op.Operation != ast.Query && args.R.Method == http.MethodGet {
	//	return ctx, nil, nil, gqlerror.List{gqlerror.Errorf("GET requests only allow query operations")}
	//}

	return reqParams, func(response *graphql.Response) {
		b, err := json.Marshal(response)
		if err != nil {
			panic(err)
		}
		w.Write(b)
	}
}

func jsonDecode(r io.Reader, val interface{}) error {
	dec := json.NewDecoder(r)
	dec.UseNumber()
	return dec.Decode(val)
}
