package dataloader

import (
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"time"
)

func NewCustomerLoader(keys []int) *CustomerLoader {
	fmt.Println("SELECT * FROM customer")

	time.Sleep(5 * time.Millisecond)
	customers := make([]*Customer, 0, len(keys))
	for _, key := range keys {
		customers = append(customers, &Customer{
			ID: key,
			Name: strconv.Itoa(key),
			AddressID: rand.Int() % 10,
		})
	}

	loader := &CustomerLoader{
		Customers:    customers,
		CustomersById: map[int]*Customer{},
	}

	for _, customer := range customers {
		loader.CustomersById[customer.ID] = customer
		customer.Loader = loader
	}

	return loader
}

type CustomerLoader struct {
	Customers []*Customer
	CustomersById map[int]*Customer

	onceAddresses sync.Once
	onceOrders sync.Once
}

func (l *CustomerLoader) LoadAddresses() {
	l.onceAddresses.Do(func() {
		ids := make([]int, 0, len(l.Customers))
		for _, customer := range l.Customers {
			ids = append(ids, customer.AddressID)
		}

		addressLoader := NewAddressLoader(ids)
		for _, customer := range l.Customers {
			customer.ResolvedAddress = addressLoader.AddressesById[customer.AddressID]
		}
	})
}

func (l *CustomerLoader) LoadOrders() {
	l.onceOrders.Do(func() {
		ids := make([]int, 0, len(l.Customers))
		for _, customer := range l.Customers {
			ids = append(ids, customer.ID)
		}

		orderLoader := NewOrderSliceLoader(ids)
		for _, customer := range l.Customers {
			customer.ResolvedOrders = orderLoader.OrdersByCustomerId[customer.ID]
		}
	})
}
