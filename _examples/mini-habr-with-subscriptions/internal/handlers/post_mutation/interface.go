package postmutation

import (
	"github.com/google/uuid"
	"github.com/nabishec/ozon_habr_api/internal/model"
)

type PostMutImp interface {
	AddPost(newPost *model.NewPost) (*model.Post, error)
	UpdateEnableCommentToPost(postID int64, authorID uuid.UUID, commentsEnabled bool) (*model.Post, error)
}
