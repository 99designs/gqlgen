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
	"github.com/99designs/gqlgen/handler"
)

func main() {
	http.Handle("/", handler.Playground("File Upload Demo", "/query"))
	resolver := getResolver()
	exec := fileupload.NewExecutableSchema(fileupload.Config{Resolvers: resolver})

	var mb int64 = 1 << 20
	uploadMaxMemory := handler.UploadMaxMemory(32 * mb)
	uploadMaxSize := handler.UploadMaxSize(50 * mb)

	http.Handle("/query", handler.GraphQL(exec, uploadMaxMemory, uploadMaxSize))
	log.Print("connect to http://localhost:8087/ for GraphQL playground")
	log.Fatal(http.ListenAndServe(":8087", nil))
}

func getResolver() *fileupload.Resolver {
	resolver := &fileupload.Resolver{
		SingleUploadFunc: func(ctx context.Context, file graphql.Upload) (*model.File, error) {
			content, err := ioutil.ReadAll(file.File)
			if err != nil {
				return nil, err
			}
			return &model.File{
				ID:      1,
				Name:    file.Filename,
				Content: string(content),
			}, nil
		},
		SingleUploadWithPayloadFunc: func(ctx context.Context, req model.UploadFile) (*model.File, error) {
			content, err := ioutil.ReadAll(req.File.File)
			if err != nil {
				return nil, err
			}
			return &model.File{
				ID:      1,
				Name:    req.File.Filename,
				Content: string(content),
			}, nil
		},
		MultipleUploadFunc: func(ctx context.Context, files []graphql.Upload) ([]model.File, error) {
			if len(files) == 0 {
				return nil, errors.New("empty list")
			}
			var resp []model.File
			for i := range files {
				content, err := ioutil.ReadAll(files[i].File)
				if err != nil {
					return []model.File{}, err
				}
				resp = append(resp, model.File{
					ID:      i + 1,
					Name:    files[i].Filename,
					Content: string(content),
				})
			}
			return resp, nil
		},
		MultipleUploadWithPayloadFunc: func(ctx context.Context, req []model.UploadFile) ([]model.File, error) {
			if len(req) == 0 {
				return nil, errors.New("empty list")
			}
			var resp []model.File
			for i := range req {
				content, err := ioutil.ReadAll(req[i].File.File)
				if err != nil {
					return []model.File{}, err
				}
				resp = append(resp, model.File{
					ID:      i + 1,
					Name:    req[i].File.Filename,
					Content: string(content),
				})
			}
			return resp, nil
		},
	}
	return resolver
}
