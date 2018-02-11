//go:generate gorunpkg github.com/vektah/gqlgen -out generated.go

package dataloader

import (
	"context"
	"fmt"
	"time"
)

type Resolver struct{}

func (r *Resolver) Customer_address(ctx context.Context, it *Customer) (*Address, error) {
	return ctxLoaders(ctx).addressByID.Load(it.addressID)
}

func (r *Resolver) Customer_orders(ctx context.Context, it *Customer) ([]Order, error) {
	return ctxLoaders(ctx).ordersByCustomer.Load(it.ID)
}

func (r *Resolver) Order_items(ctx context.Context, it *Order) ([]Item, error) {
	return ctxLoaders(ctx).itemsByOrder.Load(it.ID)
}

func (r *Resolver) Query_customers(ctx context.Context) ([]Customer, error) {
	fmt.Println("SELECT * FROM customer")

	time.Sleep(5 * time.Millisecond)

	return []Customer{
		{ID: 1, Name: "Bob", addressID: 1},
		{ID: 2, Name: "Alice", addressID: 3},
		{ID: 3, Name: "Eve", addressID: 4},
	}, nil
}
