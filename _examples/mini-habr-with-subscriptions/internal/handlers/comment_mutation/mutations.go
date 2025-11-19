package commentmutation

import (
	"fmt"

	"github.com/gqlgen/_examples/mini-habr-with-subscriptions/internal/model"
	"github.com/gqlgen/_examples/mini-habr-with-subscriptions/internal/pkg/errs"
	"github.com/rs/zerolog/log"
)

type CommentMutation struct {
	commentMutationImp CommentMutationImp
}

func NewCommentMutation(commentMutationImp CommentMutationImp) *CommentMutation {
	return &CommentMutation{commentMutationImp: commentMutationImp}
}

func (h *CommentMutation) AddComment(
	postID int64,
	newComment *model.NewComment,
) (*model.Comment, error) {
	op := "internal.handlers.commentmutation.AddComment()"

	log.Debug().Msgf("%s start", op)

	if len(newComment.Text) > 2000 || len(newComment.Text) <= 0 {
		return nil, errs.ErrIncorrectCommentLength
	}

	comment, err := h.commentMutationImp.AddComment(postID, newComment)
	if err != nil {
		if err == errs.ErrPostNotExist || err == errs.ErrParentCommentNotExist ||
			err == errs.ErrCommentsNotEnabled {
			return nil, err
		}
		return nil, fmt.Errorf("%s:%w", op, err)
	}

	log.Debug().Msgf("%s end", op)
	return comment, nil
}
