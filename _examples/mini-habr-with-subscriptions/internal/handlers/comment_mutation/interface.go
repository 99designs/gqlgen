package commentmutation

import (
	"github.com/gqlgen/_examples/mini-habr-with-subscriptions/internal/model"
)

type CommentMutationImp interface {
	AddComment(postID int64, newComment *model.NewComment) (*model.Comment, error)
}
