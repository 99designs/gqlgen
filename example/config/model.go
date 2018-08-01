package config

import "fmt"

type User struct {
	ID                  string
	FirstName, LastName string
}

func (user *User) FullName() string {
	return fmt.Sprintf("%s %s", user.FirstName, user.LastName)
}
