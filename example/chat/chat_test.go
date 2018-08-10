package chat

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/handler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChatSubscriptions(t *testing.T) {
	srv := httptest.NewServer(handler.GraphQL(NewExecutableSchema(New())))
	c := client.New(srv.URL)

	sub := c.Websocket(`subscription { messageAdded(roomName:"#gophers") { text createdBy } }`)
	defer sub.Close()

	go func() {
		var resp interface{}
		time.Sleep(10 * time.Millisecond)
		err := c.Post(`mutation { 
				a:post(text:"Hello!", roomName:"#gophers", username:"vektah") { id } 
				b:post(text:"Whats up?", roomName:"#gophers", username:"vektah") { id } 
			}`, &resp)
		assert.NoError(t, err)
	}()

	var msg struct {
		resp struct {
			MessageAdded struct {
				Text      string
				CreatedBy string
			}
		}
		err error
	}

	msg.err = sub.Next(&msg.resp)
	require.NoError(t, msg.err, "sub.Next")
	require.Equal(t, "Hello!", msg.resp.MessageAdded.Text)
	require.Equal(t, "vektah", msg.resp.MessageAdded.CreatedBy)

	msg.err = sub.Next(&msg.resp)
	require.NoError(t, msg.err, "sub.Next")
	require.Equal(t, "Whats up?", msg.resp.MessageAdded.Text)
	require.Equal(t, "vektah", msg.resp.MessageAdded.CreatedBy)
}
