//go:generate go run ../../testdata/gqlgen.go -stub stubs.go
package fileupload

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"

	"github.com/99designs/gqlgen/example/fileupload/model"
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/stretchr/testify/require"
)

func TestFileUpload(t *testing.T) {
	client := http.Client{}

	t.Run("valid single file upload", func(t *testing.T) {
		resolver := &Stub{}
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
		srv := httptest.NewServer(handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: resolver})))
		defer srv.Close()

		operations := `{ "query": "mutation ($file: Upload!) { singleUpload(file: $file) { id, name, content, contentType } }", "variables": { "file": null } }`
		mapData := `{ "0": ["variables.file"] }`
		files := []file{
			{
				mapKey:      "0",
				name:        "a.txt",
				content:     "test",
				contentType: "text/plain",
			},
		}
		req := createUploadRequest(t, srv.URL, operations, mapData, files)

		resp, err := client.Do(req)
		require.Nil(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		responseBody, err := ioutil.ReadAll(resp.Body)
		require.Nil(t, err)
		responseString := string(responseBody)
		require.Equal(t, `{"data":{"singleUpload":{"id":1,"name":"a.txt","content":"test","contentType":"text/plain"}}}`, responseString)
		err = resp.Body.Close()
		require.Nil(t, err)
	})

	t.Run("valid single file upload with payload", func(t *testing.T) {
		resolver := &Stub{}
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
		srv := httptest.NewServer(handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: resolver})))
		defer srv.Close()

		operations := `{ "query": "mutation ($req: UploadFile!) { singleUploadWithPayload(req: $req) { id, name, content, contentType } }", "variables": { "req": {"file": null, "id": 1 } } }`
		mapData := `{ "0": ["variables.req.file"] }`
		files := []file{
			{
				mapKey:      "0",
				name:        "a.txt",
				content:     "test",
				contentType: "text/plain",
			},
		}
		req := createUploadRequest(t, srv.URL, operations, mapData, files)

		resp, err := client.Do(req)
		require.Nil(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		responseBody, err := ioutil.ReadAll(resp.Body)
		require.Nil(t, err)
		require.Equal(t, `{"data":{"singleUploadWithPayload":{"id":1,"name":"a.txt","content":"test","contentType":"text/plain"}}}`, string(responseBody))
		err = resp.Body.Close()
		require.Nil(t, err)
	})

	t.Run("valid file list upload", func(t *testing.T) {
		resolver := &Stub{}
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
		srv := httptest.NewServer(handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: resolver})))
		defer srv.Close()

		operations := `{ "query": "mutation($files: [Upload!]!) { multipleUpload(files: $files) { id, name, content, contentType } }", "variables": { "files": [null, null] } }`
		mapData := `{ "0": ["variables.files.0"], "1": ["variables.files.1"] }`
		files := []file{
			{
				mapKey:      "0",
				name:        "a.txt",
				content:     "test1",
				contentType: "text/plain",
			},
			{
				mapKey:      "1",
				name:        "b.txt",
				content:     "test2",
				contentType: "text/plain",
			},
		}
		req := createUploadRequest(t, srv.URL, operations, mapData, files)

		resp, err := client.Do(req)
		require.Nil(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		responseBody, err := ioutil.ReadAll(resp.Body)
		require.Nil(t, err)
		require.Equal(t, `{"data":{"multipleUpload":[{"id":1,"name":"a.txt","content":"test1","contentType":"text/plain"},{"id":2,"name":"b.txt","content":"test2","contentType":"text/plain"}]}}`, string(responseBody))
		err = resp.Body.Close()
		require.Nil(t, err)
	})

	t.Run("valid file list upload with payload", func(t *testing.T) {
		resolver := &Stub{}
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
		srv := httptest.NewServer(handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: resolver})))
		defer srv.Close()

		operations := `{ "query": "mutation($req: [UploadFile!]!) { multipleUploadWithPayload(req: $req) { id, name, content, contentType } }", "variables": { "req": [ { "id": 1, "file": null }, { "id": 2, "file": null } ] } }`
		mapData := `{ "0": ["variables.req.0.file"], "1": ["variables.req.1.file"] }`
		files := []file{
			{
				mapKey:      "0",
				name:        "a.txt",
				content:     "test1",
				contentType: "text/plain",
			},
			{
				mapKey:      "1",
				name:        "b.txt",
				content:     "test2",
				contentType: "text/plain",
			},
		}
		req := createUploadRequest(t, srv.URL, operations, mapData, files)

		resp, err := client.Do(req)
		require.Nil(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		responseBody, err := ioutil.ReadAll(resp.Body)
		require.Nil(t, err)
		require.Equal(t, `{"data":{"multipleUploadWithPayload":[{"id":1,"name":"a.txt","content":"test1","contentType":"text/plain"},{"id":2,"name":"b.txt","content":"test2","contentType":"text/plain"}]}}`, string(responseBody))
		err = resp.Body.Close()
		require.Nil(t, err)
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

		operations := `{ "query": "mutation($req: [UploadFile!]!) { multipleUploadWithPayload(req: $req) { id, name, content, contentType } }", "variables": { "req": [ { "id": 1, "file": null }, { "id": 2, "file": null } ] } }`
		mapData := `{ "0": ["variables.req.0.file", "variables.req.1.file"] }`
		files := []file{
			{
				mapKey:      "0",
				name:        "a.txt",
				content:     "test1",
				contentType: "text/plain",
			},
		}

		test := func(uploadMaxMemory int64) {
			hndlr := handler.New(NewExecutableSchema(Config{Resolvers: resolver}))
			hndlr.AddTransport(transport.MultipartForm{MaxMemory: uploadMaxMemory})

			srv := httptest.NewServer(hndlr)
			defer srv.Close()
			req := createUploadRequest(t, srv.URL, operations, mapData, files)
			resp, err := client.Do(req)
			require.Nil(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
			responseBody, err := ioutil.ReadAll(resp.Body)
			require.Nil(t, err)
			require.Equal(t, `{"data":{"multipleUploadWithPayload":[{"id":1,"name":"a.txt","content":"test1","contentType":"text/plain"},{"id":2,"name":"a.txt","content":"test1","contentType":"text/plain"}]}}`, string(responseBody))
			err = resp.Body.Close()
			require.Nil(t, err)
		}

		t.Run("payload smaller than UploadMaxMemory, stored in memory", func(t *testing.T) {
			test(5000)
		})

		t.Run("payload bigger than UploadMaxMemory, persisted to disk", func(t *testing.T) {
			test(2)
		})
	})
}

type file struct {
	mapKey      string
	name        string
	content     string
	contentType string
}

func createUploadRequest(t *testing.T, url, operations, mapData string, files []file) *http.Request {
	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)

	err := bodyWriter.WriteField("operations", operations)
	require.NoError(t, err)

	err = bodyWriter.WriteField("map", mapData)
	require.NoError(t, err)

	for i := range files {
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, files[i].mapKey, files[i].name))
		h.Set("Content-Type", files[i].contentType)
		ff, err := bodyWriter.CreatePart(h)
		require.NoError(t, err)
		_, err = ff.Write([]byte(files[i].content))
		require.NoError(t, err)
	}
	err = bodyWriter.Close()
	require.NoError(t, err)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/graphql", url), bodyBuf)
	require.NoError(t, err)

	req.Header.Set("Content-Type", bodyWriter.FormDataContentType())
	return req
}
