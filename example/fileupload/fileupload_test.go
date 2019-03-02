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

	"github.com/99designs/gqlgen/handler"
	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/example/fileupload/model"
)

func TestFileUpload(t *testing.T) {

	t.Run("valid singleUpload", func(t *testing.T) {
		resolver := &Resolver{
			SingleUploadFunc: func(ctx context.Context, file model.Upload) (*model.File, error) {
				require.NotNil(t, file)
				require.NotNil(t, file.File)
				content, err := ioutil.ReadAll(file.File)
				require.Nil(t, err)
				require.Equal(t, string(content), "test")

				return &model.File{
					ID: 1,
				}, nil
			},
		}
		srv := httptest.NewServer(handler.GraphQL(NewExecutableSchema(Config{Resolvers: resolver})))
		defer srv.Close()

		bodyBuf := &bytes.Buffer{}
		bodyWriter := multipart.NewWriter(bodyBuf)
		err := bodyWriter.WriteField("operations", `{ "query": "mutation ($file: Upload!) { singleUpload(file: $file) { id } }", "variables": { "file": null } }`)
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
		require.Equal(t, `{"data":{"singleUpload":{"id":1}}}`, string(responseBody))
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
					ID: 1,
				}, nil
			},
		}
		srv := httptest.NewServer(handler.GraphQL(NewExecutableSchema(Config{Resolvers: resolver})))
		defer srv.Close()

		bodyBuf := &bytes.Buffer{}
		bodyWriter := multipart.NewWriter(bodyBuf)
		err := bodyWriter.WriteField("operations", `{ "query": "mutation ($req: UploadFile!) { singleUploadWithPayload(req: $req) { id } }", "variables": { "req": {"file": null, "id": 1 } } }`)
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
		require.Equal(t, `{"data":{"singleUploadWithPayload":{"id":1}}}`, string(responseBody))
	})

	t.Run("valid file list upload", func(t *testing.T) {
		resolver := &Resolver{
			MultipleUploadFunc: func(ctx context.Context, files []model.Upload) ([]model.File, error) {
				require.Len(t, files, 2)
				for i := range files {
					require.NotNil(t, files[i].File)
					content, err := ioutil.ReadAll(files[i].File)
					require.Nil(t, err)
					require.Equal(t, string(content), "test")
				}
				return []model.File{
					{ID: 1},
					{ID: 2},
				}, nil
			},
		}
		srv := httptest.NewServer(handler.GraphQL(NewExecutableSchema(Config{Resolvers: resolver})))
		defer srv.Close()

		bodyBuf := &bytes.Buffer{}
		bodyWriter := multipart.NewWriter(bodyBuf)
		err := bodyWriter.WriteField("operations", `{ "query": "mutation($files: [Upload!]!) { multipleUpload(files: $files) { id } }", "variables": { "files": [null, null] } }`)
		require.NoError(t, err)
		err = bodyWriter.WriteField("map", `{ "0": ["variables.files.0"], "1": ["variables.files.1"] }`)
		require.NoError(t, err)
		w0, err := bodyWriter.CreateFormFile("0", "a.txt")
		require.NoError(t, err)
		_, err = w0.Write([]byte("test"))
		require.NoError(t, err)
		w1, err := bodyWriter.CreateFormFile("1", "b.txt")
		require.NoError(t, err)
		_, err = w1.Write([]byte("test"))
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
		require.Equal(t, `{"data":{"multipleUpload":[{"id":1},{"id":2}]}}`, string(responseBody))
	})

}
