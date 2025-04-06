package storage

import (
	"github.com/google/uuid"
	"github.com/nabishec/ozon_habr_api/internal/model"
)

type StorageImp interface {
	AddPost(newPost *model.NewPost) (*model.Post, error)
	AddComment(postID int64, newComment *model.NewComment) (*model.Comment, error)
	UpdateEnableCommentToPost(postID int64, authorID uuid.UUID, commentsEnabled bool) (*model.Post, error)
	GetAllPosts() ([]*model.Post, error)
	GetPost(postID int64) (*model.Post, error)
	GetCommentsBranch(postID int64, path string) ([]*model.Comment, error)
	GetCommentPath(parentID int64) (string, error)
}
