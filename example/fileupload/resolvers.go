//go:generate go run ../../testdata/gqlgen.go

package fileupload

import (
	"context"
	"fmt"

	"github.com/99designs/gqlgen/example/fileupload/model"
	"github.com/99designs/gqlgen/graphql"
)

type Resolver struct {
	SingleUploadFunc              func(ctx context.Context, file graphql.Upload) (*model.File, error)
	SingleUploadWithPayloadFunc   func(ctx context.Context, req model.UploadFile) (*model.File, error)
	MultipleUploadFunc            func(ctx context.Context, files []graphql.Upload) ([]model.File, error)
	MultipleUploadWithPayloadFunc func(ctx context.Context, req []model.UploadFile) ([]model.File, error)
}

func (r *Resolver) Mutation() MutationResolver {
	return &mutationResolver{r}
}
func (r *Resolver) Query() QueryResolver {
	return &queryResolver{r}
}

type mutationResolver struct{ *Resolver }

func (r *mutationResolver) SingleUpload(ctx context.Context, file graphql.Upload) (*model.File, error) {
	if r.SingleUploadFunc != nil {
		return r.SingleUploadFunc(ctx, file)
	}
	return nil, fmt.Errorf("not implemented")
}
func (r *mutationResolver) SingleUploadWithPayload(ctx context.Context, req model.UploadFile) (*model.File, error) {
	if r.SingleUploadWithPayloadFunc != nil {
		return r.SingleUploadWithPayloadFunc(ctx, req)
	}
	return nil, fmt.Errorf("not implemented")
}
func (r *mutationResolver) MultipleUpload(ctx context.Context, files []graphql.Upload) ([]model.File, error) {
	if r.MultipleUploadFunc != nil {
		return r.MultipleUploadFunc(ctx, files)
	}
	return nil, fmt.Errorf("not implemented")
}

func (r *mutationResolver) MultipleUploadWithPayload(ctx context.Context, req []model.UploadFile) ([]model.File, error) {
	if r.MultipleUploadWithPayloadFunc != nil {
		return r.MultipleUploadWithPayloadFunc(ctx, req)
	}
	return nil, fmt.Errorf("not implemented")
}

type queryResolver struct{ *Resolver }

func (r *queryResolver) Empty(ctx context.Context) (string, error) {
	return "", fmt.Errorf("not implemented")
}
