package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/vektah/gqlgen/example/chat"
	"github.com/vektah/gqlgen/handler"
)

func main() {
	http.Handle("/", handler.Playground("Todo", "/query"))
	http.Handle("/query", handler.GraphQL(chat.MakeExecutableSchema(chat.New()),
		handler.WebsocketUpgrader(websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		})),
	)
	log.Fatal(http.ListenAndServe(":8085", nil))
}
