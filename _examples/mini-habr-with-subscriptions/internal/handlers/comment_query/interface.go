package commentquery

import (
	"github.com/nabishec/ozon_habr_api/internal/model"
)

type CommentQueryImp interface {
	GetCommentsBranch(postID int64, path string) ([]*model.Comment, error)
	GetCommentPath(parentID int64) (string, error)
}
