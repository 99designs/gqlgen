package dataloader

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func NewAddressLoader(keys []int) *AddressLoader {
	var keySql []string
	for _, key := range keys {
		keySql = append(keySql, strconv.Itoa(key))
	}

	fmt.Printf("SELECT * FROM address WHERE id IN (%s)\n", strings.Join(keySql, ","))
	time.Sleep(5 * time.Millisecond)

	addresses := make([]*Address, len(keys))
	for i, key := range keys {
		addresses[i] = &Address{ID: key, Street: "home street", Country: "hometon " + strconv.Itoa(key)}
	}

	loader := &AddressLoader{
		AddressesById: map[int]*Address{},
	}

	for _, address := range addresses {
		loader.AddressesById[address.ID] = address
		address.Loader = loader
	}
	return loader
}

type AddressLoader struct {
	Addresses []*Address
	AddressesById map[int]*Address
}
