//go:generate rm -f resolver.go
//go:generate go run ../../../testdata/gqlgen.go -config gqlgen.yml -stub stub.go

package childfielddedup

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
)

func TestChildFieldDedup_QueryWithNestedUserFields(t *testing.T) {
	resolvers := &Stub{}
	resolvers.QueryResolver.Book = func(ctx context.Context, id string) (*Book, error) {
		name := "Alice"
		return &Book{
			ID:    "book-1",
			Title: "Test Book",
			Author: &Author{
				ID:    "author-1",
				Name:  "Bob",
				Email: "bob@example.com",
			},
			Reviewer: &User{
				ID:     "user-1",
				Name:   name,
				Email:  "alice@example.com",
				Role:   "reviewer",
				Status: "active",
			},
		}, nil
	}

	srv := handler.New(NewExecutableSchema(Config{Resolvers: resolvers}))
	srv.AddTransport(transport.POST{})
	c := client.New(srv)

	t.Run("nested user fields via shared childFields", func(t *testing.T) {
		var resp struct {
			Book struct {
				ID     string
				Title  string
				Author struct {
					ID   string
					Name string
				}
				Reviewer struct {
					ID     string
					Name   string
					Email  string
					Role   string
					Status string
				}
			}
		}
		c.MustPost(`query {
			book(id: "book-1") {
				id
				title
				author { id name }
				reviewer { id name email role status }
			}
		}`, &resp)
		require.Equal(t, "book-1", resp.Book.ID)
		require.Equal(t, "Test Book", resp.Book.Title)
		require.Equal(t, "Bob", resp.Book.Author.Name)
		require.Equal(t, "user-1", resp.Book.Reviewer.ID)
		require.Equal(t, "alice@example.com", resp.Book.Reviewer.Email)
	})

	t.Run("multiple entities referencing User type", func(t *testing.T) {
		resolvers.QueryResolver.Comment = func(ctx context.Context, id string) (*Comment, error) {
			return &Comment{
				ID:   "comment-1",
				Text: "Great!",
				Commenter: &User{
					ID:     "user-2",
					Name:   "Charlie",
					Email:  "charlie@example.com",
					Role:   "commenter",
					Status: "active",
				},
			}, nil
		}

		var resp struct {
			Comment struct {
				ID        string
				Text      string
				Commenter struct {
					ID   string
					Name string
				}
			}
		}
		c.MustPost(`query {
			comment(id: "comment-1") {
				id
				text
				commenter { id name }
			}
		}`, &resp)
		require.Equal(t, "comment-1", resp.Comment.ID)
		require.Equal(t, "Charlie", resp.Comment.Commenter.Name)
	})
}
