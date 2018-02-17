package chat

import (
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlgen/client"
	"github.com/vektah/gqlgen/handler"
)

func TestChat(t *testing.T) {
	srv := httptest.NewServer(handler.GraphQL(MakeExecutableSchema(New())))
	c := client.New(srv.URL)
	var wg sync.WaitGroup
	wg.Add(1)

	t.Run("subscribe to chat events", func(t *testing.T) {
		t.Parallel()

		sub := c.Websocket(`subscription { messageAdded(roomName:"#gophers") { text createdBy } }`)
		defer sub.Close()

		wg.Done()
		var resp struct {
			MessageAdded struct {
				Text      string
				CreatedBy string
			}
		}
		require.NoError(t, sub.Next(&resp))
		require.Equal(t, "Hello!", resp.MessageAdded.Text)
		require.Equal(t, "vektah", resp.MessageAdded.CreatedBy)

		require.NoError(t, sub.Next(&resp))
		require.Equal(t, "Whats up?", resp.MessageAdded.Text)
		require.Equal(t, "vektah", resp.MessageAdded.CreatedBy)
	})

	t.Run("post two messages", func(t *testing.T) {
		t.Parallel()

		wg.Wait()
		var resp interface{}
		c.MustPost(`mutation { 
			a:post(text:"Hello!", roomName:"#gophers", username:"vektah") { id } 
			b:post(text:"Whats up?", roomName:"#gophers", username:"vektah") { id } 
		}`, &resp)
	})

}
