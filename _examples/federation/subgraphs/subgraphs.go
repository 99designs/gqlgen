package subgraphs

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"

	"golang.org/x/sync/errgroup"

	"github.com/john-markham/gqlgen/graphql"
	"github.com/john-markham/gqlgen/graphql/handler"
	"github.com/john-markham/gqlgen/graphql/handler/debug"
	"github.com/john-markham/gqlgen/graphql/handler/extension"
	"github.com/john-markham/gqlgen/graphql/handler/transport"
	"github.com/john-markham/gqlgen/graphql/playground"
)

type Config struct {
	EnableDebug bool
}

type Subgraphs struct {
	servers []*http.Server
}

type SubgraphConfig struct {
	Name   string
	Schema graphql.ExecutableSchema
	Port   string
}

func (s *Subgraphs) Shutdown(ctx context.Context) error {
	for _, srv := range s.servers {
		if err := srv.Shutdown(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (s *Subgraphs) ListenAndServe(ctx context.Context) error {
	group, _ := errgroup.WithContext(ctx)
	for _, srv := range s.servers {
		group.Go(func() error {
			err := srv.ListenAndServe()
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Printf("error listening and serving: %v", err)
				return err
			}
			return nil
		})
	}
	return group.Wait()
}

func newServer(name, port string, schema graphql.ExecutableSchema) *http.Server {
	if port == "" {
		panic(fmt.Errorf("port for %s is empty", name))
	}
	srv := handler.New(schema)
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.Use(extension.Introspection{})
	srv.Use(&debug.Tracer{})
	mux := http.NewServeMux()
	mux.Handle("/", playground.Handler("GraphQL playground", "/query"))
	mux.Handle("/query", srv)
	return &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}
}

func New(ctx context.Context, subgraphs ...SubgraphConfig) (*Subgraphs, error) {
	servers := make([]*http.Server, len(subgraphs))
	for i, config := range subgraphs {
		servers[i] = newServer(config.Name, config.Port, config.Schema)
	}

	return &Subgraphs{
		servers: servers,
	}, nil
}
