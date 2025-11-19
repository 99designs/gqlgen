package graph

import (
	commentmutation "github.com/gqlgen/_examples/mini-habr-with-subscriptions/internal/handlers/comment_mutation"
	commentquery "github.com/gqlgen/_examples/mini-habr-with-subscriptions/internal/handlers/comment_query"
	postmutation "github.com/gqlgen/_examples/mini-habr-with-subscriptions/internal/handlers/post_mutation"
	postquery "github.com/gqlgen/_examples/mini-habr-with-subscriptions/internal/handlers/post_query"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	PostMutation    *postmutation.PostMutation
	PostQuery       *postquery.PostQuery
	CommentMutation *commentmutation.CommentMutation
	CommentQuery    *commentquery.CommentQuery
	Subscribers     *Subscribers
}

func NewResolver(
	postMutation *postmutation.PostMutation,
	postQuery *postquery.PostQuery,
	commentMutation *commentmutation.CommentMutation,
	commentQuery *commentquery.CommentQuery,
) *Resolver {
	return &Resolver{
		PostMutation:    postMutation,
		PostQuery:       postQuery,
		CommentMutation: commentMutation,
		CommentQuery:    commentQuery,
		Subscribers:     NewSubscribers(),
	}
}
