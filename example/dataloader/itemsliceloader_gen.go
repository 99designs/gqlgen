package dataloader

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

func NewItemSliceLoader(keys []int) *ItemSliceLoader {
	var keySql []string
	for _, key := range keys {
		keySql = append(keySql, strconv.Itoa(key))
	}

	fmt.Printf("SELECT * FROM items JOIN item_order WHERE item_order.order_id IN (%s)\n", strings.Join(keySql, ","))
	time.Sleep(5 * time.Millisecond)

	items := make([]*Item, 0, len(keys))
	for i := range keys {
		items = append(items, []*Item{
			{OrderID: i, Name: "item " + strconv.Itoa(rand.Int()%20+20)},
			{OrderID: i, Name: "item " + strconv.Itoa(rand.Int()%20+20)},
		}...)
	}

	loader := &ItemSliceLoader{
		Items: items,
		ItemsByOrderId: map[int][]*Item{},
	}

	for _, item := range items {
		loader.ItemsByOrderId[item.OrderID] = append(loader.ItemsByOrderId[item.OrderID], item)
	}

	return loader
}

type ItemSliceLoader struct {
	Items []*Item
	ItemsByOrderId map[int][]*Item
}
