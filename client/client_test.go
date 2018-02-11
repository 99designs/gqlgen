package client_test

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vektah/gqlgen/client"

	"github.com/stretchr/testify/require"
)

func TestClient(t *testing.T) {
	h := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			panic(err)
		}
		require.Equal(t, `{"query":"user(id:$id){name}","variables":{"id":1}}`, string(b))

		err = json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"name": "bob",
			},
		})
		if err != nil {
			panic(err)
		}
	}))

	c := client.New(h.URL)

	var resp struct {
		Name string
	}

	c.MustPost("user(id:$id){name}", &resp, client.Var("id", 1))

	require.Equal(t, "bob", resp.Name)
}
