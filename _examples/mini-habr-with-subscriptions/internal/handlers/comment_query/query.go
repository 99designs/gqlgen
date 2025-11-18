package commentquery

import (
	"fmt"

	"github.com/gqlgen/_examples/mini-habr-with-subscriptions/internal/model"
	"github.com/gqlgen/_examples/mini-habr-with-subscriptions/internal/pkg/errs"
	"github.com/rs/zerolog/log"
)

type CommentQuery struct {
	commentQueryImp CommentQueryImp
}

func NewCommentQuery(commentQueryImp CommentQueryImp) *CommentQuery {
	return &CommentQuery{commentQueryImp: commentQueryImp}
}

func (h *CommentQuery) GetCommentsBranchToPost(
	postID int64,
	path string,
) ([]*model.Comment, error) {
	op := "internal.handlers.commentquery.GetCommentsBranchToPost()"

	log.Debug().Msgf("%s start", op)

	comments, err := h.commentQueryImp.GetCommentsBranch(postID, path)
	if err != nil {
		if err == errs.ErrCommentsNotExist || err == errs.ErrPathNotExist {
			return nil, err
		}
		return nil, fmt.Errorf("%s:%w", op, err)
	}

	log.Debug().Msgf("%s end", op)
	return comments, nil
}

func (h *CommentQuery) GetPathToComments(parentID int64) (string, error) {
	op := "internal.handlers.commentquery.GetPathToComments()"

	log.Debug().Msgf("%s start", op)

	path, err := h.commentQueryImp.GetCommentPath(parentID)
	if err != nil {
		if err == errs.ErrCommentsNotExist {
			return "", err
		}
		return "", fmt.Errorf("%s:%w", op, err)
	}

	log.Debug().Msgf("%s start", op)

	return path, nil
}
