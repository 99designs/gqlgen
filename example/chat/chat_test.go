package chat

import (
	"testing"
	"time"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChatSubscriptions(t *testing.T) {
	c := client.New(handler.NewDefaultServer(NewExecutableSchema(New())))

	sub := c.Websocket(`subscription @user(username:"vektah") { messageAdded(roomName:"#gophers") { text createdBy } }`)
	defer sub.Close()

	go func() {
		var resp interface{}
		time.Sleep(10 * time.Millisecond)
		err := c.Post(`mutation { 
				a:post(text:"Hello!", roomName:"#gophers", username:"vektah") { id } 
				b:post(text:"Hello Vektah!", roomName:"#gophers", username:"andrey") { id } 
				c:post(text:"Whats up?", roomName:"#gophers", username:"vektah") { id } 
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
