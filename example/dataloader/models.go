package dataloader

import "time"

type Customer struct {
	ID        int
	Name      string
	addressID int
}

type Address struct {
	ID      int
	Street  string
	Country string
}

type Order struct {
	ID     int
	Date   time.Time
	Amount float64
}

type Item struct {
	Name string
}
