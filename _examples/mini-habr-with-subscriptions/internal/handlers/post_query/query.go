package postquery

import (
	"fmt"

	"github.com/gqlgen/_examples/mini-habr-with-subscriptions/internal/model"
	"github.com/gqlgen/_examples/mini-habr-with-subscriptions/internal/pkg/errs"
	"github.com/rs/zerolog/log"
)

type PostQuery struct {
	postQueryImp PostQueryImp
}

func NewPostQuery(postQueryImp PostQueryImp) *PostQuery {
	return &PostQuery{postQueryImp: postQueryImp}
}

func (h *PostQuery) GetAllPosts() ([]*model.Post, error) {
	op := "internal.handlers.postquery.GetAllPosts()"

	log.Debug().Msgf("%s start", op)

	posts, err := h.postQueryImp.GetAllPosts()

	if err != nil {
		if err == errs.ErrPostsNotExist {
			return nil, err
		}
		return nil, fmt.Errorf("%s:%w", op, err)
	}

	log.Debug().Msgf("%s end", op)
	return posts, nil
}

func (h *PostQuery) GetPost(postID int64) (*model.Post, error) {
	op := "internal.handlers.postquery.GetPostWithComment()"

	log.Debug().Msgf("%s start", op)

	post, err := h.postQueryImp.GetPost(postID)

	if err != nil {
		if err == errs.ErrPostNotExist {
			return nil, err
		}
		return nil, fmt.Errorf("%s:%w", op, err)
	}

	log.Debug().Msgf("%s end", op)
	return post, nil
}
