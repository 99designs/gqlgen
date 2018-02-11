package main

import (
	"log"
	"net/http"

	"github.com/vektah/graphql-go"
	"github.com/vektah/graphql-go/example/dataloader"
)

func main() {
	http.Handle("/", graphql.GraphiqlHandler("Dataloader", "/query"))

	http.Handle("/query", dataloader.LoaderMiddleware(graphql.Handler(dataloader.NewExecutor(&dataloader.Resolver{}))))

	log.Fatal(http.ListenAndServe(":8082", nil))
}
