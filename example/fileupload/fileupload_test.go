package fileupload

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/99designs/gqlgen/example/fileupload/model"
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/handler"
	"github.com/stretchr/testify/require"
)

func TestFileUpload(t *testing.T) {

	t.Run("valid single file upload", func(t *testing.T) {
		resolver := &Resolver{
			SingleUploadFunc: func(ctx context.Context, file graphql.Upload) (*model.File, error) {
				require.NotNil(t, file)
				require.NotNil(t, file.File)
				content, err := ioutil.ReadAll(file.File)
				require.Nil(t, err)
				require.Equal(t, string(content), "test")

				return &model.File{
					ID:      1,
					Name:    file.Filename,
					Content: string(content),
				}, nil
			},
		}
		srv := httptest.NewServer(handler.GraphQL(NewExecutableSchema(Config{Resolvers: resolver})))
		defer srv.Close()

		bodyBuf := &bytes.Buffer{}
		bodyWriter := multipart.NewWriter(bodyBuf)
		err := bodyWriter.WriteField("operations", `{ "query": "mutation ($file: Upload!) { singleUpload(file: $file) { id, name, content } }", "variables": { "file": null } }`)
		require.NoError(t, err)
		err = bodyWriter.WriteField("map", `{ "0": ["variables.file"] }`)
		require.NoError(t, err)
		w, err := bodyWriter.CreateFormFile("0", "a.txt")
		require.NoError(t, err)
		_, err = w.Write([]byte("test"))
		require.NoError(t, err)
		err = bodyWriter.Close()
		require.NoError(t, err)

		contentType := bodyWriter.FormDataContentType()

		resp, err := http.Post(fmt.Sprintf("%s/graphql", srv.URL), contentType, bodyBuf)
		require.Nil(t, err)
		defer func() {
			_ = resp.Body.Close()
		}()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		responseBody, err := ioutil.ReadAll(resp.Body)
		require.Nil(t, err)
		responseString := string(responseBody)
		require.Equal(t, `{"data":{"singleUpload":{"id":1,"name":"a.txt","content":"test"}}}`, responseString)
	})

	t.Run("valid single file upload with payload", func(t *testing.T) {
		resolver := &Resolver{
			SingleUploadWithPayloadFunc: func(ctx context.Context, req model.UploadFile) (*model.File, error) {
				require.Equal(t, req.ID, 1)
				require.NotNil(t, req.File)
				require.NotNil(t, req.File.File)
				content, err := ioutil.ReadAll(req.File.File)
				require.Nil(t, err)
				require.Equal(t, string(content), "test")

				return &model.File{
					ID:      1,
					Name:    req.File.Filename,
					Content: string(content),
				}, nil
			},
		}
		srv := httptest.NewServer(handler.GraphQL(NewExecutableSchema(Config{Resolvers: resolver})))
		defer srv.Close()

		bodyBuf := &bytes.Buffer{}
		bodyWriter := multipart.NewWriter(bodyBuf)
		err := bodyWriter.WriteField("operations", `{ "query": "mutation ($req: UploadFile!) { singleUploadWithPayload(req: $req) { id, name, content } }", "variables": { "req": {"file": null, "id": 1 } } }`)
		require.NoError(t, err)
		err = bodyWriter.WriteField("map", `{ "0": ["variables.req.file"] }`)
		require.NoError(t, err)
		w, err := bodyWriter.CreateFormFile("0", "a.txt")
		require.NoError(t, err)
		_, err = w.Write([]byte("test"))
		require.NoError(t, err)
		err = bodyWriter.Close()
		require.NoError(t, err)

		contentType := bodyWriter.FormDataContentType()

		resp, err := http.Post(fmt.Sprintf("%s/graphql", srv.URL), contentType, bodyBuf)
		require.Nil(t, err)
		defer func() {
			_ = resp.Body.Close()
		}()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		responseBody, err := ioutil.ReadAll(resp.Body)
		require.Nil(t, err)
		require.Equal(t, `{"data":{"singleUploadWithPayload":{"id":1,"name":"a.txt","content":"test"}}}`, string(responseBody))
	})

	t.Run("valid file list upload", func(t *testing.T) {
		resolver := &Resolver{
			MultipleUploadFunc: func(ctx context.Context, files []graphql.Upload) ([]model.File, error) {
				require.Len(t, files, 2)
				var contents []string
				var resp []model.File
				for i := range files {
					require.NotNil(t, files[i].File)
					content, err := ioutil.ReadAll(files[i].File)
					require.Nil(t, err)
					contents = append(contents, string(content))
					resp = append(resp, model.File{
						ID:      i + 1,
						Name:    files[i].Filename,
						Content: string(content),
					})
				}
				require.ElementsMatch(t, []string{"test1", "test2"}, contents)
				return resp, nil
			},
		}
		srv := httptest.NewServer(handler.GraphQL(NewExecutableSchema(Config{Resolvers: resolver})))
		defer srv.Close()

		bodyBuf := &bytes.Buffer{}
		bodyWriter := multipart.NewWriter(bodyBuf)
		err := bodyWriter.WriteField("operations", `{ "query": "mutation($files: [Upload!]!) { multipleUpload(files: $files) { id, name, content } }", "variables": { "files": [null, null] } }`)
		require.NoError(t, err)
		err = bodyWriter.WriteField("map", `{ "0": ["variables.files.0"], "1": ["variables.files.1"] }`)
		require.NoError(t, err)
		w0, err := bodyWriter.CreateFormFile("0", "a.txt")
		require.NoError(t, err)
		_, err = w0.Write([]byte("test1"))
		require.NoError(t, err)
		w1, err := bodyWriter.CreateFormFile("1", "b.txt")
		require.NoError(t, err)
		_, err = w1.Write([]byte("test2"))
		require.NoError(t, err)
		err = bodyWriter.Close()
		require.NoError(t, err)

		contentType := bodyWriter.FormDataContentType()

		resp, err := http.Post(fmt.Sprintf("%s/graphql", srv.URL), contentType, bodyBuf)
		require.Nil(t, err)
		defer func() {
			_ = resp.Body.Close()
		}()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		responseBody, err := ioutil.ReadAll(resp.Body)
		require.Nil(t, err)
		require.Equal(t, `{"data":{"multipleUpload":[{"id":1,"name":"a.txt","content":"test1"},{"id":2,"name":"b.txt","content":"test2"}]}}`, string(responseBody))
	})

	t.Run("valid file list upload with payload", func(t *testing.T) {
		resolver := &Resolver{
			MultipleUploadWithPayloadFunc: func(ctx context.Context, req []model.UploadFile) ([]model.File, error) {
				require.Len(t, req, 2)
				var ids []int
				var contents []string
				var resp []model.File
				for i := range req {
					require.NotNil(t, req[i].File)
					require.NotNil(t, req[i].File.File)
					content, err := ioutil.ReadAll(req[i].File.File)
					require.Nil(t, err)
					ids = append(ids, req[i].ID)
					contents = append(contents, string(content))
					resp = append(resp, model.File{
						ID:      i + 1,
						Name:    req[i].File.Filename,
						Content: string(content),
					})
				}
				require.ElementsMatch(t, []int{1, 2}, ids)
				require.ElementsMatch(t, []string{"test1", "test2"}, contents)
				return resp, nil
			},
		}
		srv := httptest.NewServer(handler.GraphQL(NewExecutableSchema(Config{Resolvers: resolver})))
		defer srv.Close()

		bodyBuf := &bytes.Buffer{}
		bodyWriter := multipart.NewWriter(bodyBuf)
		err := bodyWriter.WriteField("operations", `{ "query": "mutation($req: [UploadFile!]!) { multipleUploadWithPayload(req: $req) { id, name, content } }", "variables": { "req": [ { "id": 1, "file": null }, { "id": 2, "file": null } ] } }`)
		require.NoError(t, err)
		err = bodyWriter.WriteField("map", `{ "0": ["variables.req.0.file"], "1": ["variables.req.1.file"] }`)
		require.NoError(t, err)
		w0, err := bodyWriter.CreateFormFile("0", "a.txt")
		require.NoError(t, err)
		_, err = w0.Write([]byte("test1"))
		require.NoError(t, err)
		w1, err := bodyWriter.CreateFormFile("1", "b.txt")
		require.NoError(t, err)
		_, err = w1.Write([]byte("test2"))
		require.NoError(t, err)
		err = bodyWriter.Close()
		require.NoError(t, err)

		contentType := bodyWriter.FormDataContentType()

		resp, err := http.Post(fmt.Sprintf("%s/graphql", srv.URL), contentType, bodyBuf)
		require.Nil(t, err)
		defer func() {
			_ = resp.Body.Close()
		}()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		responseBody, err := ioutil.ReadAll(resp.Body)
		require.Nil(t, err)
		require.Equal(t, `{"data":{"multipleUploadWithPayload":[{"id":1,"name":"a.txt","content":"test1"},{"id":2,"name":"b.txt","content":"test2"}]}}`, string(responseBody))
	})

}
