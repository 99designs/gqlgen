package postmutation

import (
	"github.com/google/uuid"
	"github.com/gqlgen/_examples/mini-habr-with-subscriptions/internal/model"
)

type PostMutImp interface {
	AddPost(newPost *model.NewPost) (*model.Post, error)
	UpdateEnableCommentToPost(postID int64, authorID uuid.UUID, commentsEnabled bool) (*model.Post, error)
}
