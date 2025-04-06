package commentmutation

import (
	"github.com/nabishec/ozon_habr_api/internal/model"
)

type CommentMutationImp interface {
	AddComment(postID int64, newComment *model.NewComment) (*model.Comment, error)
}
