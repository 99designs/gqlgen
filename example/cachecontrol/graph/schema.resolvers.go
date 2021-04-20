// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.
package graph

import (
	"context"
	"time"

	"github.com/99designs/gqlgen/example/cachecontrol/graph/generated"
	"github.com/99designs/gqlgen/example/cachecontrol/graph/model"
	"github.com/99designs/gqlgen/graphql/handler/cache"
)

func (r *commentResolver) Post(ctx context.Context, obj *model.Comment) (*model.Post, error) {
	cache.SetHint(ctx, cache.ScopePublic, 10*time.Second)
	return obj.Post, nil
}

func (r *postResolver) Votes(ctx context.Context, obj *model.Post) (int, error) {
	cache.SetHint(ctx, cache.ScopePublic, 30*time.Second)
	return obj.Votes, nil
}

func (r *postResolver) Comments(ctx context.Context, obj *model.Post) ([]*model.Comment, error) {
	cache.SetHint(ctx, cache.ScopePublic, 1000*time.Second)
	return obj.Comments, nil
}

func (r *postResolver) ReadByCurrentUser(ctx context.Context, obj *model.Post) (bool, error) {
	cache.SetHint(ctx, cache.ScopePrivate, 2*time.Second)
	return obj.ReadByCurrentUser, nil
}

func (r *queryResolver) LatestPost(ctx context.Context) (*model.Post, error) {
	cache.SetHint(ctx, cache.ScopePublic, 10*time.Second)
	post := &model.Post{
		ID:                10,
		Votes:             3,
		ReadByCurrentUser: false,
	}
	post.Comments = []*model.Comment{
		{
			Post: post,
			Text: "Comment 1",
		},
		{
			Post: post,
			Text: "Comment 2",
		},
	}
	return post, nil
}

func (r *queryResolver) Post(ctx context.Context, id int) (*model.Post, error) {
	cache.SetHint(ctx, cache.ScopePublic, 30*time.Second)

	post := &model.Post{
		ID:                1,
		Votes:             10,
		ReadByCurrentUser: false,
	}
	post.Comments = []*model.Comment{
		{
			Post: post,
			Text: "Comment 1",
		},
	}
	return post, nil
}

func (r *Resolver) Comment() generated.CommentResolver { return &commentResolver{r} }
func (r *Resolver) Post() generated.PostResolver       { return &postResolver{r} }
func (r *Resolver) Query() generated.QueryResolver     { return &queryResolver{r} }

type commentResolver struct{ *Resolver }
type postResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
