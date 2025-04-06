package postquery

import (
	"github.com/nabishec/ozon_habr_api/internal/model"
)

type PostQueryImp interface {
	GetAllPosts() ([]*model.Post, error)
	GetPost(postID int64) (*model.Post, error)
}
