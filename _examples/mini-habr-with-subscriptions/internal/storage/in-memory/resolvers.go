package inmemory

import (
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gqlgen/_examples/mini-habr-with-subscriptions/internal/model"
	"github.com/gqlgen/_examples/mini-habr-with-subscriptions/internal/pkg/errs"
	"github.com/rs/zerolog/log"
)

type Storage struct {
	postsLastIndex   int64
	commentLastIndex int64
	comments         map[int64][]*model.Comment
	commentPath      map[int64]string
	repliesByPath    map[string][]*model.Comment
	posts            map[int64]*model.Post
}

func NewStorage() *Storage {
	return &Storage{
		postsLastIndex:   0,
		commentLastIndex: 0,
		comments:         make(map[int64][]*model.Comment), // root comments
		commentPath:      make(map[int64]string),
		repliesByPath:    make(map[string][]*model.Comment),
		posts:            make(map[int64]*model.Post),
	}
}

func (r *Storage) AddPost(newPost *model.NewPost) (*model.Post, error) {
	op := "internal.storage.inmemory.AddPost()"

	log.Debug().Msgf("%s start", op)
	post := &model.Post{
		AuthorID:        newPost.AuthorID,
		Title:           newPost.Title,
		Text:            newPost.Text,
		CommentsEnabled: newPost.CommentsEnabled,
		CreateDate:      time.Now(),
	}

	postID := r.postsLastIndex + 1
	r.postsLastIndex += 1
	post.ID = postID
	r.posts[postID] = post

	log.Debug().Msgf("%s end", op)
	return post, nil
}

func (r *Storage) AddComment(postID int64, newComment *model.NewComment) (*model.Comment, error) {
	op := "internal.storage.inmemory.AddComment()"

	log.Debug().Msgf("%s start", op)

	comment := &model.Comment{
		AuthorID:   newComment.AuthorID,
		PostID:     postID,
		ParentID:   newComment.ParentID,
		Text:       newComment.Text,
		CreateDate: time.Now(),
	}

	post, ok := r.posts[postID]
	if !ok {
		return nil, errs.ErrPostNotExist
	}

	if !post.CommentsEnabled {
		return nil, errs.ErrCommentsNotEnabled
	}

	commentID := r.commentLastIndex + 1
	r.commentLastIndex += 1
	comment.ID = commentID
	var parentPath string
	if comment.ParentID != nil {
		if parentCommentPath, ok := r.commentPath[*comment.ParentID]; ok {
			parentPath = parentCommentPath
		} else {
			return nil, errs.ErrParentCommentNotExist
		}
		r.repliesByPath[parentPath] = append(r.repliesByPath[parentPath], comment)
		parentPath += "."

	} else {
		r.comments[postID] = append(r.comments[postID], comment)
	}

	path := parentPath + strconv.FormatInt(comment.ID, 10)
	comment.Path = path

	r.commentPath[commentID] = path

	log.Debug().Msgf("%s end", op)
	return comment, nil
}

func (r *Storage) UpdateEnableCommentToPost(postID int64, authorID uuid.UUID, commentsEnabled bool) (*model.Post, error) {
	op := "internal.storage.inmemory.UpdateEnableCommentToPost()"

	log.Debug().Msgf("%s start", op)

	post, ok := r.posts[postID]
	if !ok {
		return nil, errs.ErrPostNotExist
	}

	if post.AuthorID != authorID {
		return nil, errs.ErrUnauthorizedAccess
	}

	post.CommentsEnabled = commentsEnabled

	log.Debug().Msgf("%s end", op)
	return post, nil
}

func (r *Storage) GetAllPosts() ([]*model.Post, error) {
	op := "internal.storage.inmemory.GetAllPosts()"

	log.Debug().Msgf("%s start", op)

	var posts []*model.Post
	for _, v := range r.posts {
		posts = append(posts, v)
	}
	if len(posts) == 0 {
		return nil, errs.ErrPostsNotExist
	}

	log.Debug().Msgf("%s end", op)

	return posts, nil
}

func (r *Storage) GetPost(postID int64) (*model.Post, error) {
	op := "internal.storage.inmemory.GetPost()"

	log.Debug().Msgf("%s start", op)

	post, ok := r.posts[postID]

	if !ok {
		return nil, errs.ErrPostNotExist
	}

	log.Debug().Msgf("%s end", op)

	return post, nil
}

func (r *Storage) GetCommentsBranch(postID int64, path string) ([]*model.Comment, error) {
	op := "internal.storage.inmemory.GetCommentsBranch()"

	log.Debug().Msgf("%s start", op)

	if path == "" {
		if v, ok := r.comments[postID]; ok {
			return v, nil
		}
		return nil, errs.ErrCommentsNotExist
	}

	comments, ok := r.repliesByPath[path]
	if !ok {
		return nil, errs.ErrPathNotExist
	}

	if len(comments) == 0 {
		return nil, errs.ErrCommentsNotExist
	}

	return comments, nil

}

func (r *Storage) GetCommentPath(commentID int64) (string, error) {
	op := "internal.storage.inmemory.GetCommentPath()"

	log.Debug().Msgf("%s start", op)

	path, ok := r.commentPath[commentID]

	if !ok {
		return "", errs.ErrCommentsNotExist
	}

	log.Debug().Msgf("%s end", op)
	return path, nil
}
