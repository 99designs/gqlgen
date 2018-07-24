package chat

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlgen/client"
	"github.com/vektah/gqlgen/handler"
)

func TestChat(t *testing.T) {
	srv := httptest.NewServer(handler.GraphQL(NewExecutableSchema(New())))
	c := client.New(srv.URL)

	t.Run("subscribe to chat events", func(t *testing.T) {
		t.Parallel()

		sub := c.Websocket(`subscription { messageAdded(roomName:"#gophers") { text createdBy } }`)
		defer sub.Close()

		postErrCh := make(chan error)
		go func() {
			var resp interface{}
			// can't call t.Fatal from separate goroutine, so we return to the err chan for later
			err := c.Post(`mutation { 
				a:post(text:"Hello!", roomName:"#gophers", username:"vektah") { id } 
				b:post(text:"Whats up?", roomName:"#gophers", username:"vektah") { id } 
			}`, &resp)
			if err != nil {
				// only push this error if non-nil
				postErrCh <- err
			}
		}()

		type resp struct {
			MessageAdded struct {
				Text      string
				CreatedBy string
			}
		}

		// Contains the result of a `sub.Next` call
		type subMsg struct {
			resp
			err error
		}

		subCh := make(chan subMsg)
		go func() {
			var msg subMsg

			msg.err = sub.Next(&msg.resp)
			subCh <- msg

			msg.err = sub.Next(&msg.resp)
			subCh <- msg
		}()

		var m subMsg
		// Either can fail, and results in a failed test.
		//
		// Using a select prevents us from hanging the test
		// in the event of a failure and instead reports
		// back immediately.
		select {
		case m = <-subCh:
		case err := <-postErrCh:
			require.NoError(t, err, "post 2 messages")
		}
		require.NoError(t, m.err, "sub.Next")
		require.Equal(t, "Hello!", m.resp.MessageAdded.Text)
		require.Equal(t, "vektah", m.resp.MessageAdded.CreatedBy)

		m = <-subCh
		require.NoError(t, m.err, "sub.Next")
		require.Equal(t, "Whats up?", m.resp.MessageAdded.Text)
		require.Equal(t, "vektah", m.resp.MessageAdded.CreatedBy)
	})

}
