package transport

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/99designs/gqlgen/graphql"
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

//func writeError(enableMsgpackEncoding bool, w io.Writer, msg string) {
//	writeResponse(enableMsgpackEncoding, w, &graphql.Response{Errors: gqlerror.List{{Message: msg}}})
//}
//
//func writeErrorf(enableMsgpackEncoding bool, w io.Writer, format string, args ...interface{}) {
//	writeResponse(enableMsgpackEncoding, w, &graphql.Response{Errors: gqlerror.List{{Message: fmt.Sprintf(format, args...)}}})
//}
//
//func writeGraphqlError(enableMsgpackEncoding bool, w io.Writer, err ...*gqlerror.Error) {
//	writeResponse(enableMsgpackEncoding, w, &graphql.Response{Errors: err})
//}

//func decode(enableMsgpackEncoding bool, r io.Reader, val interface{}) error {
//	if enableMsgpackEncoding {
//		return msgpackDecode(r, val)
//	}
//
//	return jsonDecode(r, val)
//}
