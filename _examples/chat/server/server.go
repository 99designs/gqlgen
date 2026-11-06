package main

import (
	"log"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/jub0bs/cors"

	"github.com/99designs/gqlgen/_examples/chat"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
)

func main() {
	cors, err := cors.NewMiddleware(cors.Config{
		Origins:        []string{"http://localhost:3000"},
		Credentialed:   true,
		RequestHeaders: []string{"Accept", "Content-Type", "X-Requested-With"},
	})
	if err != nil {
		log.Fatal(err)
	}

	srv := handler.New(chat.NewExecutableSchema(chat.New()))

	srv.AddTransport(transport.SSE{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
		Implementation: transport.CoderWebsocketImplementation{
			AcceptOptions: websocket.AcceptOptions{
				OriginPatterns: []string{"http://localhost:3000"},
			},
		},
	})
	srv.Use(extension.Introspection{})

	http.Handle("/", playground.Handler("Todo", "/query"))
	http.Handle("/query", cors.Wrap(srv))

	log.Fatal(http.ListenAndServe(":8085", nil))
}
