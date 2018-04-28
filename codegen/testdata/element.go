package testdata

import (
	"context"
	"errors"
	"time"
)

type Element struct {
	ID int
}

type ElementResolver struct{}

func (r *ElementResolver) Query_path(ctx context.Context) ([]Element, error) {
	return []Element{{1}, {2}, {3}, {4}}, nil
}

func (r *ElementResolver) Element_child(ctx context.Context, obj *Element) (Element, error) {
	return Element{obj.ID * 10}, nil
}

func (r *ElementResolver) Element_error(ctx context.Context, obj *Element, message *string) (bool, error) {
	// A silly hack to make the result order stable
	time.Sleep(time.Duration(obj.ID) * 10 * time.Millisecond)

	if message != nil {
		return true, errors.New(*message)
	}
	return false, nil
}
