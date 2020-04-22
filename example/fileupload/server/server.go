package main

import (
	"context"
	"errors"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/99designs/gqlgen/example/fileupload"
	"github.com/99designs/gqlgen/example/fileupload/model"
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
)

func main() {
	var mb int64 = 1 << 20
	resolver := getResolver()

	srv := handler.New(fileupload.NewExecutableSchema(fileupload.Config{Resolvers: resolver}))
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{
		MaxMemory:     32 * mb,
		MaxUploadSize: 50 * mb,
	})
	http.Handle("/query", srv)
	http.Handle("/", playground.Handler("File Upload Demo", "/query"))

	log.Print("connect to http://localhost:8087/ for GraphQL playground")
	log.Fatal(http.ListenAndServe(":8087", nil))
}

func getResolver() *fileupload.Stub {
	resolver := &fileupload.Stub{}

	resolver.MutationResolver.SingleUpload = func(ctx context.Context, file graphql.Upload) (*model.File, error) {
		content, err := ioutil.ReadAll(file.File)
		if err != nil {
			return nil, err
		}
		return &model.File{
			ID:          1,
			Name:        file.Filename,
			Content:     string(content),
			ContentType: file.ContentType,
		}, nil
	}
	resolver.MutationResolver.SingleUploadWithPayload = func(ctx context.Context, req model.UploadFile) (*model.File, error) {
		content, err := ioutil.ReadAll(req.File.File)
		if err != nil {
			return nil, err
		}
		return &model.File{
			ID:          1,
			Name:        req.File.Filename,
			Content:     string(content),
			ContentType: req.File.ContentType,
		}, nil
	}
	resolver.MutationResolver.MultipleUpload = func(ctx context.Context, files []*graphql.Upload) ([]*model.File, error) {
		if len(files) == 0 {
			return nil, errors.New("empty list")
		}
		var resp []*model.File
		for i, file := range files {
			content, err := ioutil.ReadAll(file.File)
			if err != nil {
				return []*model.File{}, err
			}
			resp = append(resp, &model.File{
				ID:          i + 1,
				Name:        file.Filename,
				Content:     string(content),
				ContentType: file.ContentType,
			})
		}
		return resp, nil
	}
	resolver.MutationResolver.MultipleUploadWithPayload = func(ctx context.Context, reqs []*model.UploadFile) ([]*model.File, error) {
		if len(reqs) == 0 {
			return nil, errors.New("empty list")
		}
		var resp []*model.File
		for i, req := range reqs {
			content, err := ioutil.ReadAll(req.File.File)
			if err != nil {
				return []*model.File{}, err
			}
			resp = append(resp, &model.File{
				ID:          i + 1,
				Name:        req.File.Filename,
				Content:     string(content),
				ContentType: req.File.ContentType,
			})
		}
		return resp, nil
	}
	resolver.QueryResolver.Empty = func(ctx context.Context) (s string, err error) {
		return "empty", nil
	}
	return resolver
}
