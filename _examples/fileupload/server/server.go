package main

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"

	"github.com/99designs/gqlgen/graphql/playground"

	"github.com/99designs/gqlgen/_examples/fileupload"
	"github.com/99designs/gqlgen/_examples/fileupload/model"
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
)

func main() {
	http.Handle("/", playground.Handler("File Upload Demo", "/query"))
	resolver := getResolver()

	var mb int64 = 1 << 20

	srv := handler.NewDefaultServer(fileupload.NewExecutableSchema(fileupload.Config{Resolvers: resolver}))
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{
		MaxMemory:     32 * mb,
		MaxUploadSize: 50 * mb,
	})
	srv.Use(extension.Introspection{})

	http.Handle("/query", srv)
	log.Print("connect to http://localhost:8087/ for GraphQL playground")
	log.Fatal(http.ListenAndServe(":8087", nil))
}

func getResolver() *fileupload.Stub {
	resolver := &fileupload.Stub{}

	resolver.MutationResolver.SingleUpload = func(ctx context.Context, file graphql.Upload) (*model.File, error) {
		content, err := io.ReadAll(file.File)
		if err != nil {
			return nil, err
		}
		return &model.File{
			ID:      1,
			Name:    file.Filename,
			Content: string(content),
		}, nil
	}
	resolver.MutationResolver.SingleUploadWithPayload = func(ctx context.Context, req model.UploadFile) (*model.File, error) {
		content, err := io.ReadAll(req.File.File)
		if err != nil {
			return nil, err
		}
		return &model.File{
			ID:      1,
			Name:    req.File.Filename,
			Content: string(content),
		}, nil
	}
	resolver.MutationResolver.MultipleUpload = func(ctx context.Context, files []*graphql.Upload) ([]*model.File, error) {
		if len(files) == 0 {
			return nil, errors.New("empty list")
		}
		var resp []*model.File
		for i := range files {
			content, err := io.ReadAll(files[i].File)
			if err != nil {
				return []*model.File{}, err
			}
			resp = append(resp, &model.File{
				ID:      i + 1,
				Name:    files[i].Filename,
				Content: string(content),
			})
		}
		return resp, nil
	}
	resolver.MutationResolver.MultipleUploadWithPayload = func(ctx context.Context, req []*model.UploadFile) ([]*model.File, error) {
		if len(req) == 0 {
			return nil, errors.New("empty list")
		}
		var resp []*model.File
		for i := range req {
			content, err := io.ReadAll(req[i].File.File)
			if err != nil {
				return []*model.File{}, err
			}
			resp = append(resp, &model.File{
				ID:      i + 1,
				Name:    req[i].File.Filename,
				Content: string(content),
			})
		}
		return resp, nil
	}
	return resolver
}
