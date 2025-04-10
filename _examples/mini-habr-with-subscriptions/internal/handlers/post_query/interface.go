package postquery

import (
	"github.com/gqlgen/_examples/mini-habr-with-subscriptions/internal/model"
)

type PostQueryImp interface {
	GetAllPosts() ([]*model.Post, error)
	GetPost(postID int64) (*model.Post, error)
}
