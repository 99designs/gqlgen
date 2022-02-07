//go:generate go run ../../testdata/gqlgen.go -stub stubs.go
package fileupload

import (
	"context"
	"io"
	"io/ioutil"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/99designs/gqlgen/_examples/fileupload/model"
	gqlclient "github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/stretchr/testify/require"
)

func TestFileUpload(t *testing.T) {
	resolver := &Stub{}
	srv := httptest.NewServer(handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: resolver})))
	defer srv.Close()
	gql := gqlclient.New(srv.Config.Handler, gqlclient.Path("/graphql"))

	aTxtFile, _ := ioutil.TempFile(os.TempDir(), "a.txt")
	defer os.Remove(aTxtFile.Name())
	aTxtFile.WriteString(`test`)

	a1TxtFile, _ := ioutil.TempFile(os.TempDir(), "a.txt")
	b1TxtFile, _ := ioutil.TempFile(os.TempDir(), "b.txt")
	defer os.Remove(a1TxtFile.Name())
	defer os.Remove(b1TxtFile.Name())
	a1TxtFile.WriteString(`test1`)
	b1TxtFile.WriteString(`test2`)

	t.Run("valid single file upload", func(t *testing.T) {
		resolver.MutationResolver.SingleUpload = func(ctx context.Context, file graphql.Upload) (*model.File, error) {
			require.NotNil(t, file)
			require.NotNil(t, file.File)
			content, err := ioutil.ReadAll(file.File)
			require.Nil(t, err)
			require.Equal(t, string(content), "test")

			return &model.File{
				ID:          1,
				Name:        file.Filename,
				Content:     string(content),
				ContentType: file.ContentType,
			}, nil
		}

		mutation := `mutation ($file: Upload!) {
			singleUpload(file: $file) {
				id
				name
				content
				contentType
			}
		}`
		var result struct {
			SingleUpload *model.File
		}

		err := gql.Post(mutation, &result, gqlclient.Var("file", aTxtFile), gqlclient.WithFiles())
		require.Nil(t, err)
		require.Equal(t, 1, result.SingleUpload.ID)
		require.Contains(t, result.SingleUpload.Name, "a.txt")
		require.Equal(t, "test", result.SingleUpload.Content)
		require.Equal(t, "text/plain; charset=utf-8", result.SingleUpload.ContentType)
	})

	t.Run("valid single file upload with payload", func(t *testing.T) {
		resolver.MutationResolver.SingleUploadWithPayload = func(ctx context.Context, req model.UploadFile) (*model.File, error) {
			require.Equal(t, req.ID, 1)
			require.NotNil(t, req.File)
			require.NotNil(t, req.File.File)
			content, err := ioutil.ReadAll(req.File.File)
			require.Nil(t, err)
			require.Equal(t, string(content), "test")

			return &model.File{
				ID:          1,
				Name:        req.File.Filename,
				Content:     string(content),
				ContentType: req.File.ContentType,
			}, nil
		}

		mutation := `mutation ($req: UploadFile!) {
			singleUploadWithPayload(req: $req) {
				id
				name
				content
				contentType
			}
		}`
		var result struct {
			SingleUploadWithPayload *model.File
		}

		err := gql.Post(mutation, &result, gqlclient.Var("req", map[string]interface{}{"id": 1, "file": aTxtFile}), gqlclient.WithFiles())
		require.Nil(t, err)
		require.Equal(t, 1, result.SingleUploadWithPayload.ID)
		require.Contains(t, result.SingleUploadWithPayload.Name, "a.txt")
		require.Equal(t, "test", result.SingleUploadWithPayload.Content)
		require.Equal(t, "text/plain; charset=utf-8", result.SingleUploadWithPayload.ContentType)
	})

	t.Run("valid file list upload", func(t *testing.T) {
		resolver.MutationResolver.MultipleUpload = func(ctx context.Context, files []*graphql.Upload) ([]*model.File, error) {
			require.Len(t, files, 2)
			var contents []string
			var resp []*model.File
			for i := range files {
				require.NotNil(t, files[i].File)
				content, err := ioutil.ReadAll(files[i].File)
				require.Nil(t, err)
				contents = append(contents, string(content))
				resp = append(resp, &model.File{
					ID:          i + 1,
					Name:        files[i].Filename,
					Content:     string(content),
					ContentType: files[i].ContentType,
				})
			}
			require.ElementsMatch(t, []string{"test1", "test2"}, contents)
			return resp, nil
		}

		mutation := `mutation($files: [Upload!]!) {
			multipleUpload(files: $files) {
				id
				name
				content
				contentType
			}
		}`
		var result struct {
			MultipleUpload []*model.File
		}

		err := gql.Post(mutation, &result, gqlclient.Var("files", []*os.File{a1TxtFile, b1TxtFile}), gqlclient.WithFiles())
		require.Nil(t, err)
		require.Equal(t, 1, result.MultipleUpload[0].ID)
		require.Equal(t, 2, result.MultipleUpload[1].ID)
		for _, mu := range result.MultipleUpload {
			if mu.Name == "a.txt" {
				require.Equal(t, "test1", mu.Content)
			}
			if mu.Name == "b.txt" {
				require.Equal(t, "test2", mu.Content)
			}
			require.Equal(t, "text/plain; charset=utf-8", mu.ContentType)
		}
	})

	t.Run("valid file list upload with payload", func(t *testing.T) {
		resolver.MutationResolver.MultipleUploadWithPayload = func(ctx context.Context, req []*model.UploadFile) ([]*model.File, error) {
			require.Len(t, req, 2)
			var ids []int
			var contents []string
			var resp []*model.File
			for i := range req {
				require.NotNil(t, req[i].File)
				require.NotNil(t, req[i].File.File)
				content, err := ioutil.ReadAll(req[i].File.File)
				require.Nil(t, err)
				ids = append(ids, req[i].ID)
				contents = append(contents, string(content))
				resp = append(resp, &model.File{
					ID:          i + 1,
					Name:        req[i].File.Filename,
					Content:     string(content),
					ContentType: req[i].File.ContentType,
				})
			}
			require.ElementsMatch(t, []int{1, 2}, ids)
			require.ElementsMatch(t, []string{"test1", "test2"}, contents)
			return resp, nil
		}

		mutation := `mutation($req: [UploadFile!]!) {
			multipleUploadWithPayload(req: $req) {
				id
				name
				content
				contentType
			}
		}`
		var result struct {
			MultipleUploadWithPayload []*model.File
		}

		err := gql.Post(mutation, &result, gqlclient.Var("req", []map[string]interface{}{
			{"id": 1, "file": a1TxtFile},
			{"id": 2, "file": b1TxtFile},
		}), gqlclient.WithFiles())
		require.Nil(t, err)
		require.Equal(t, 1, result.MultipleUploadWithPayload[0].ID)
		require.Equal(t, 2, result.MultipleUploadWithPayload[1].ID)
		for _, mu := range result.MultipleUploadWithPayload {
			if mu.Name == "a.txt" {
				require.Equal(t, "test1", mu.Content)
			}
			if mu.Name == "b.txt" {
				require.Equal(t, "test2", mu.Content)
			}
			require.Equal(t, "text/plain; charset=utf-8", mu.ContentType)
		}
	})

	t.Run("valid file list upload with payload and file reuse", func(t *testing.T) {
		resolver := &Stub{}
		resolver.MutationResolver.MultipleUploadWithPayload = func(ctx context.Context, req []*model.UploadFile) ([]*model.File, error) {
			require.Len(t, req, 2)
			var ids []int
			var contents []string
			var resp []*model.File
			for i := range req {
				require.NotNil(t, req[i].File)
				require.NotNil(t, req[i].File.File)
				ids = append(ids, req[i].ID)

				var got []byte
				buf := make([]byte, 2)
				for {
					n, err := req[i].File.File.Read(buf)
					got = append(got, buf[:n]...)
					if err != nil {
						if err == io.EOF {
							break
						}
						require.Fail(t, "unexpected error while reading", err.Error())
					}
				}
				contents = append(contents, string(got))
				resp = append(resp, &model.File{
					ID:          i + 1,
					Name:        req[i].File.Filename,
					Content:     string(got),
					ContentType: req[i].File.ContentType,
				})
			}
			require.ElementsMatch(t, []int{1, 2}, ids)
			require.ElementsMatch(t, []string{"test1", "test1"}, contents)
			return resp, nil
		}

		test := func(uploadMaxMemory int64) {
			hndlr := handler.New(NewExecutableSchema(Config{Resolvers: resolver}))
			hndlr.AddTransport(transport.MultipartForm{MaxMemory: uploadMaxMemory})

			srv := httptest.NewServer(hndlr)
			defer srv.Close()
			gql := gqlclient.New(srv.Config.Handler, gqlclient.Path("/graphql"))

			mutation := `mutation($req: [UploadFile!]!) {
				multipleUploadWithPayload(req: $req) {
					id
					name
					content
					contentType
				}
			}`
			var result struct {
				MultipleUploadWithPayload []*model.File
			}

			err := gql.Post(mutation, &result, gqlclient.Var("req", []map[string]interface{}{
				{"id": 1, "file": a1TxtFile},
				{"id": 2, "file": a1TxtFile},
			}), gqlclient.WithFiles())
			require.Nil(t, err)
			require.Equal(t, 1, result.MultipleUploadWithPayload[0].ID)
			require.Contains(t, result.MultipleUploadWithPayload[0].Name, "a.txt")
			require.Equal(t, "test1", result.MultipleUploadWithPayload[0].Content)
			require.Equal(t, "text/plain; charset=utf-8", result.MultipleUploadWithPayload[0].ContentType)
			require.Equal(t, 2, result.MultipleUploadWithPayload[1].ID)
			require.Contains(t, result.MultipleUploadWithPayload[1].Name, "a.txt")
			require.Equal(t, "test1", result.MultipleUploadWithPayload[1].Content)
			require.Equal(t, "text/plain; charset=utf-8", result.MultipleUploadWithPayload[1].ContentType)
		}

		t.Run("payload smaller than UploadMaxMemory, stored in memory", func(t *testing.T) {
			test(5000)
		})

		t.Run("payload bigger than UploadMaxMemory, persisted to disk", func(t *testing.T) {
			test(2)
		})
	})
}
