package main

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/client"
)

func TestServer(t *testing.T) {
	c := client.New(new())
	actual, err := c.RawPost(`{
	  latestPost {
		id
		comments {
		  text
		  post {
			id
		  }
		}
		readByCurrentUser
	  }
	}`)
	require.NoError(t, err)

	var expected map[string]interface{}
	err = json.Unmarshal([]byte(`{
    "cacheControl": {
      "version": 1,
      "hints": [
        {
          "path": [
            "latestPost"
          ],
          "maxAge": 10,
          "scope": "PUBLIC"
        },
        {
          "path": [
            "latestPost",
            "comments"
          ],
          "maxAge": 1000,
          "scope": "PUBLIC"
        },
        {
          "path": [
            "latestPost",
            "readByCurrentUser"
          ],
          "maxAge": 2,
          "scope": "PRIVATE"
        },
        {
          "path": [
            "latestPost",
            "comments",
            1,
            "post"
          ],
          "maxAge": 10,
          "scope": "PUBLIC"
        },
        {
          "path": [
            "latestPost",
            "comments",
            0,
            "post"
          ],
          "maxAge": 10,
          "scope": "PUBLIC"
        }
      ]
    }}`), &expected)

	require.NoError(t, err)
	require.Nil(t, actual.Errors)
	require.NotNil(t, actual.Data)
	expectedCacheControl := expected["cacheControl"].(map[string]interface{})
	actualCacheControl := actual.Extensions["cacheControl"].(map[string]interface{})
	require.Equal(t, expectedCacheControl["version"], actualCacheControl["version"])
	require.ElementsMatch(t, expectedCacheControl["hints"], actualCacheControl["hints"])
}
