package graph

import (
	"github.com/gqlgen/_examples/mini-habr-with-subscriptions/graph/model"
	internalmodel "github.com/gqlgen/_examples/mini-habr-with-subscriptions/internal/model"
	"github.com/gqlgen/_examples/mini-habr-with-subscriptions/internal/pkg/cursor"
)

const defaultFirst int = 5

func newPostToInternalModel(post *model.NewPost) *internalmodel.NewPost {
	return &internalmodel.NewPost{
		AuthorID:        post.AuthorID,
		Title:           post.Title,
		Text:            post.Text,
		CommentsEnabled: post.CommentsEnabled,
	}
}

func postFromInternalModel(internalPost *internalmodel.Post) *model.Post {
	return &model.Post{
		ID:              internalPost.ID,
		AuthorID:        internalPost.AuthorID,
		Title:           internalPost.Title,
		Text:            internalPost.Text,
		CommentsEnabled: internalPost.CommentsEnabled,
		CreateDate:      internalPost.CreateDate,
	}
}

func newCommentToInternalModel(newComment *model.NewComment) *internalmodel.NewComment {
	return &internalmodel.NewComment{
		AuthorID: newComment.AuthorID,
		PostID:   newComment.PostID,
		ParentID: newComment.ParentID,
		Text:     newComment.Text,
	}
}

func paginateInternalBranch(
	internalComments []*internalmodel.Comment,
	firstInput *int32,
	after *string,
) (*model.CommentConnection, error) {
	var first int
	if firstInput == nil {
		first = defaultFirst
	} else {
		first = int(*firstInput)
	}

	edges := make([]*model.CommentEdge, 0, first)

	start := false
	var commentID int64
	var err error
	if after != nil {
		commentID, err = cursor.GetCommentID(after)
		if err != nil {
			return nil, err
		}
	} else {
		start = true
	}

	hasNextPage := false
	counter := 0
	for i, v := range internalComments {
		if v.ID == commentID && start == false {
			start = true
			continue
		}

		if start == true {
			edges = append(edges, &model.CommentEdge{
				Node:   commentFromInternalModel(v),
				Cursor: cursor.CreateCursorFromComment(v),
			})
			counter += 1
		}

		if counter == first {
			if i+1 < len(internalComments) {
				hasNextPage = true
			}
			break
		}
	}
	if start == false { // not found comment with commentID
		return &model.CommentConnection{
			Edges:    []*model.CommentEdge{},
			PageInfo: &model.PageInfo{HasNextPage: false},
		}, nil
	}
	var endCursor *string
	if len(edges) > 0 {
		endCursor = &edges[len(edges)-1].Cursor
	}

	pageInfo := &model.PageInfo{
		EndCursor:   endCursor,
		HasNextPage: hasNextPage,
	}

	return &model.CommentConnection{
		Edges:    edges,
		PageInfo: pageInfo,
	}, nil
}

func commentFromInternalModel(internalComment *internalmodel.Comment) *model.Comment {
	return &model.Comment{
		ID:         internalComment.ID,
		AuthorID:   internalComment.AuthorID,
		PostID:     internalComment.PostID,
		ParentID:   internalComment.ParentID,
		Text:       internalComment.Text,
		CreateDate: internalComment.CreateDate,
	}
}
