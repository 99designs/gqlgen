// Package model holds the Go types of the batchvalues schema of the tablemode test server,
// which both generated packages bind.
package model

// User holds its posts as values, as omit_slice_element_pointers generates them.
type User struct {
	ID    string
	Posts []Post
}

type Post struct {
	ID string
}

type Author struct {
	ID string
}
