package integration

import (
	"context"
	"errors"
	"fmt"

	"github.com/99designs/gqlgen/_examples/large-project-structure/main/graph/model"
)

type Resolver struct{}

// Implement the Tezz method that is managed by another team
func (r *Resolver) Tezz(ctx context.Context) (*model.Test, error) {
	// Can do whatever logic is needed...
	return &model.Test{ID: "external-1"}, nil
}

func (r *Resolver) GetYaSome(ctx context.Context, input *model.CustomInput) ([]*model.CustomZeekIntel, error) {
	intels := []*model.CustomZeekIntel{}

	if input.Error != nil && *input.Error {
		return intels, errors.New("error as requested")
	}

	if input.Limit != nil {
		count := int(*input.Limit)
		for i := range count {
			czi := &model.CustomZeekIntel{
				ID:         fmt.Sprintf("%d", i),
				Name:       fmt.Sprintf("external-%d", i),
				ExtraField: "let other teams resolve",
			}
			intels = append(intels, czi)
		}
	}

	return intels, nil
}

func (r *Resolver) AddIndicator(ctx context.Context, input model.IndicatorInput) (*model.Indicator, error) {
	return &model.Indicator{
		ID:            "1234",
		Indicator:     input.Indicator,
		IndicatorType: input.IndicatorType,
		MetaSource:    input.MetaSource,
	}, nil
}
