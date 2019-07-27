//go:generate go run ../../testdata/gqlgen.go

package dataloader

import (
	"context"
	"time"
)

type Customer struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	AddressID int
	ResolvedAddress *Address
	ResolvedOrders []*Order

	Loader *CustomerLoader
}
type Order struct {
	ID     int       `json:"id"`
	Date   time.Time `json:"date"`
	Amount float64   `json:"amount"`
	CustomerID int
	ResolvedItems []*Item `json:"-"`

	Loader *OrderSliceLoader
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
	obj.Loader.LoadAddresses()
	return obj.ResolvedAddress, nil
}

func (r *customerResolver) Orders(ctx context.Context, obj *Customer) ([]*Order, error) {
	obj.Loader.LoadOrders()
	return obj.ResolvedOrders, nil
}

type orderResolver struct{ *Resolver }

func (r *orderResolver) Items(ctx context.Context, obj *Order) ([]*Item, error) {
	obj.Loader.LoadItems()
	return obj.ResolvedItems, nil
}

type queryResolver struct{ *Resolver }

func (r *queryResolver) Customers(ctx context.Context) ([]*Customer, error) {
	return NewCustomerLoader([]int{1,2,3}).Customers, nil
}

// this method is here to test code generation of nested arrays
func (r *queryResolver) Torture1d(ctx context.Context, customerIds []int) ([]*Customer, error) {
	result := make([]*Customer, len(customerIds))
	loader := NewCustomerLoader(customerIds)
	for i, id := range customerIds {
		result[i] = loader.CustomersById[id]
	}
	return result, nil
}

// this method is here to test code generation of nested arrays
func (r *queryResolver) Torture2d(ctx context.Context, customerIds [][]int) ([][]*Customer, error) {
	ids := make([]int, 0, len(customerIds))
	for i := range customerIds {
		for j := range customerIds[i] {
			ids = append(ids, customerIds[i][j])
		}
	}
	loader := NewCustomerLoader(ids)

	result := make([][]*Customer, len(customerIds))

	for i := range customerIds {
		inner := make([]*Customer, len(customerIds[i]))
		for j := range customerIds[i] {
			inner[j] = loader.CustomersById[customerIds[i][j]]
		}
		result[i] = inner
	}
	return result, nil
}
