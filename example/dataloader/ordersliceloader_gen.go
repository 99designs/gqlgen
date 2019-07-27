package dataloader

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"
)


func NewOrderSliceLoader(keys []int) *OrderSliceLoader {
	var keySql []string
	for _, key := range keys {
		keySql = append(keySql, strconv.Itoa(key))
	}

	fmt.Printf("SELECT * FROM orders WHERE customer_id IN (%s)\n", strings.Join(keySql, ","))
	time.Sleep(5 * time.Millisecond)

	orders := make([]*Order, 0, len(keys))
	for i, key := range keys {
		orders = append(orders, []*Order{
			{CustomerID: key, ID: i, Amount: rand.Float64(), Date: time.Now().Add(-time.Duration(key) * time.Hour)},
			{CustomerID: key, ID: i+1, Amount: rand.Float64(), Date: time.Now().Add(-time.Duration(key) * time.Hour)},
		}...)
	}

	loader := &OrderSliceLoader{
		Orders: orders,
		OrdersByCustomerId: map[int][]*Order{},
	}

	for _, order := range orders {
		loader.OrdersByCustomerId[order.CustomerID] = append(loader.OrdersByCustomerId[order.CustomerID], order)
		order.Loader = loader
	}

	return loader
}

type OrderSliceLoader struct {
	Orders []*Order
	OrdersByCustomerId map[int][]*Order

	onceItems sync.Once
}

func (l *OrderSliceLoader) LoadItems() {
	l.onceItems.Do(func() {
		ids := make([]int, 0, len(l.Orders))
		for _, order := range l.Orders {
			ids = append(ids, order.ID)
		}

		itemLoader := NewItemSliceLoader(ids)
		for _, order := range l.Orders {
			order.ResolvedItems = itemLoader.ItemsByOrderId[order.ID]
		}
	})
}

