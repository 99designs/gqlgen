package starwars

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/john-markham/gqlgen/_examples/starwars/generated"
	"github.com/john-markham/gqlgen/graphql/handler"
	"github.com/john-markham/gqlgen/graphql/handler/transport"
)

func BenchmarkSimpleQueryNoArgs(b *testing.B) {
	server := handler.New(generated.NewExecutableSchema(NewResolver()))
	server.AddTransport(transport.POST{})

	q := `{"query":"{ search(text:\"Luke\") { ... on Human { starships { name } } } }"}`

	var body strings.Reader
	r := httptest.NewRequest("POST", "/graphql", &body)
	r.Header.Set("Content-Type", "application/json")

	b.ReportAllocs()
	b.ResetTimer()

	rec := httptest.NewRecorder()
	for i := 0; i < b.N; i++ {
		body.Reset(q)
		rec.Body.Reset()
		server.ServeHTTP(rec, r)
		if rec.Body.String() != `{"data":{"search":[{"starships":[{"name":"X-Wing"},{"name":"Imperial shuttle"}]}]}}` {
			b.Fatalf("Unexpected response: %s", rec.Body.String())
		}
	}
}
