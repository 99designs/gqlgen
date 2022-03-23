package singlefile

import (
	"context"
	"strconv"
)

type VariadicModel struct{}

type VariadicModelOption func(*VariadicModel)

func (v VariadicModel) Value(ctx context.Context, rank int, opts ...VariadicModelOption) (string, error) {
	return strconv.Itoa(rank), nil
}
