// Package model holds the Go types of the batchglobal schema of the tablemode test server,
// which both generated packages bind.
package model

type User struct {
	ID   string
	Name string
}

type Post struct {
	ID    string
	Title string
}

type Author struct {
	ID string
}
