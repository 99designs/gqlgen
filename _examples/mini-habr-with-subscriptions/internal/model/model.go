package model

import (
	"time"

	"github.com/google/uuid"
)

type Comment struct {
	ID         int64     `json:"id"                 db:"comment_id"`
	AuthorID   uuid.UUID `json:"authorID"           db:"author_id"`
	PostID     int64     `json:"postID"             db:"post_id"`
	ParentID   *int64    `json:"parentID,omitempty" db:"parent_id"`
	Path       string    `                          db:"path"`
	Text       string    `json:"text"               db:"text"`
	CreateDate time.Time `json:"createDate"         db:"create_date"`
}

type NewComment struct {
	AuthorID uuid.UUID `json:"authorID"           db:"author_id"`
	PostID   int64     `json:"postID"             db:"post_id"`
	ParentID *int64    `json:"parentID,omitempty" db:"parent_id"`
	Text     string    `json:"text"               db:"text"`
}

type NewPost struct {
	AuthorID        uuid.UUID `json:"authorID"        db:"author_id"`
	Title           string    `json:"title"           db:"title"`
	Text            string    `json:"text"            db:"text"`
	CommentsEnabled bool      `json:"commentsEnabled" db:"comments_enabled"`
}

type Post struct {
	ID              int64      `json:"id"                 db:"post_id"`
	AuthorID        uuid.UUID  `json:"authorID"           db:"author_id"`
	Title           string     `json:"title"              db:"title"`
	Text            string     `json:"text"               db:"text"`
	CommentsEnabled bool       `json:"commentsEnabled"    db:"comments_enabled"`
	Comments        []*Comment `json:"comments,omitempty"`
	CreateDate      time.Time  `json:"createDate"         db:"create_date"`
}
