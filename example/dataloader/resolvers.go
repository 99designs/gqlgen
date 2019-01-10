//go:generate go run ../../testdata/gqlgen.go

package dataloader

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

type Customer struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	AddressID int
}
type Order struct {
	ID     int       `json:"id"`
	Date   time.Time `json:"date"`
	Amount float64   `json:"amount"`
}

type Resolver struct{}

func (r *Resolver) Customer() CustomerResolver {
	return &customerResolver{r}
}

func (r *Resolver) Order() OrderResolver {
	return &orderResolver{r}
}

func (r *Resolver) Query() QueryResolver {
	return &queryResolver{r}
}

type customerResolver struct{ *Resolver }

func (r *customerResolver) Address(ctx context.Context, obj *Customer) (*Address, error) {
	return ctxLoaders(ctx).addressByID.Load(obj.AddressID)
}

func (r *customerResolver) Orders(ctx context.Context, obj *Customer) ([]Order, error) {
	return ctxLoaders(ctx).ordersByCustomer.Load(obj.ID)
}

type orderResolver struct{ *Resolver }

func (r *orderResolver) Items(ctx context.Context, obj *Order) ([]Item, error) {
	return ctxLoaders(ctx).itemsByOrder.Load(obj.ID)
}

type queryResolver struct{ *Resolver }

func (r *queryResolver) Customers(ctx context.Context) ([]Customer, error) {
	fmt.Println("SELECT * FROM customer")

	time.Sleep(5 * time.Millisecond)

	return []Customer{
		{ID: 1, Name: "Bob", AddressID: 1},
		{ID: 2, Name: "Alice", AddressID: 3},
		{ID: 3, Name: "Eve", AddressID: 4},
	}, nil
}

// this method is here to test code generation of nested arrays
func (r *queryResolver) Torture1d(ctx context.Context, customerIds []int) ([]Customer, error) {
	result := make([]Customer, len(customerIds))
	for i, id := range customerIds {
		result[i] = Customer{ID: id, Name: fmt.Sprintf("%d", i), AddressID: rand.Int() % 10}
	}
	return result, nil
}

// this method is here to test code generation of nested arrays
func (r *queryResolver) Torture2d(ctx context.Context, customerIds [][]int) ([][]Customer, error) {
	result := make([][]Customer, len(customerIds))
	for i := range customerIds {
		inner := make([]Customer, len(customerIds[i]))
		for j := range customerIds[i] {
			inner[j] = Customer{ID: customerIds[i][j], Name: fmt.Sprintf("%d %d", i, j), AddressID: rand.Int() % 10}
		}
		result[i] = inner
	}
	return result, nil
}
