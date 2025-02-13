package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/john-markham/gqlgen/_examples/deferexample"
	"github.com/john-markham/gqlgen/graphql/handler"
	"github.com/john-markham/gqlgen/graphql/handler/extension"
	"github.com/john-markham/gqlgen/graphql/handler/transport"
	"github.com/john-markham/gqlgen/graphql/playground"
	"github.com/john-markham/websocket"
	"github.com/rs/cors"
)

const defaultPort = "8080"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
		Debug:            false,
	})

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
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	})
	srv.Use(extension.Introspection{})

	http.Handle("/", playground.Handler("Todo", "/query"))
	http.Handle("/query", c.Handler(srv))

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
