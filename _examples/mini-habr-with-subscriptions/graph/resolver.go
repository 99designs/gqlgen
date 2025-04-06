package graph

import (
	commentmutation "github.com/nabishec/ozon_habr_api/internal/handlers/comment_mutation"
	commentquery "github.com/nabishec/ozon_habr_api/internal/handlers/comment_query"
	postmutation "github.com/nabishec/ozon_habr_api/internal/handlers/post_mutation"
	postquery "github.com/nabishec/ozon_habr_api/internal/handlers/post_query"
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

func NewResolver(postMutation *postmutation.PostMutation, postQuery *postquery.PostQuery, commentMutation *commentmutation.CommentMutation, commentQuery *commentquery.CommentQuery) *Resolver {
	return &Resolver{
		PostMutation:    postMutation,
		PostQuery:       postQuery,
		CommentMutation: commentMutation,
		CommentQuery:    commentQuery,
		Subscribers:     NewSubscribers(),
	}
}
