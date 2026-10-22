package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/coder/websocket"
	"github.com/jub0bs/cors"

	"github.com/99designs/gqlgen/_examples/deferexample"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
)

const defaultPort = "8080"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	cors, err := cors.NewMiddleware(cors.Config{
		Origins:        []string{"*"},
		RequestHeaders: []string{"Accept", "Content-Type", "X-Requested-With"},
	})
	if err != nil {
		log.Fatal(err)
	}

	srv := handler.New(
		deferexample.NewExecutableSchema(
			deferexample.Config{Resolvers: &deferexample.Resolver{}},
		),
	)

	srv.AddTransport(transport.SSE{})
	srv.AddTransport(transport.MultipartMixed{
		Boundary:        "graphql",
		DeliveryTimeout: time.Millisecond * 10,
	})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
		Implementation: transport.CoderWebsocketImplementation{
			AcceptOptions: websocket.AcceptOptions{
				InsecureSkipVerify: true,
			},
		},
	})
	srv.Use(extension.Introspection{})

	http.Handle("/", playground.Handler("Todo", "/query"))
	http.Handle("/query", cors.Wrap(srv))

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
