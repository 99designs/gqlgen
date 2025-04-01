package graph

import (
	"context"

	"github.com/corelight/main/graph/model"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

// Define an interface for each resolver method
type ExternalQueryResolver interface {
	// Example query resolver
	Tezz(ctx context.Context) (*model.Test, error)
	// Example query resolver with args
	GetYaSome(context.Context, *model.CustomInput) ([]*model.CustomZeekIntel, error)

	// Example mutation resolver with args
	AddIndicator(context.Context, model.IndicatorInput) (*model.Indicator, error)
}

type Resolver struct {
	ExternalQueryResolver
}
