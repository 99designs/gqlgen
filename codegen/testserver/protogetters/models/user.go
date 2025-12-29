package models

// User simulates a protobuf editions message with getter and haser methods
type User struct {
	id    string
	name  *string
	email *string
	age   int32
}

// NewUser creates a new User
func NewUser(id string, name, email *string, age int32) *User {
	return &User{
		id:    id,
		name:  name,
		email: email,
		age:   age,
	}
}

// GetId is a protobuf-style getter
func (u *User) GetId() string {
	return u.id
}

// GetName is a protobuf-style getter
func (u *User) GetName() string {
	if u.name == nil {
		return ""
	}
	return *u.name
}

// HasName is a protobuf-style haser
func (u *User) HasName() bool {
	return u.name != nil
}

// GetEmail is a protobuf-style getter
func (u *User) GetEmail() string {
	if u.email == nil {
		return ""
	}
	return *u.email
}

// HasEmail is a protobuf-style haser
func (u *User) HasEmail() bool {
	return u.email != nil
}

// GetAge is a protobuf-style getter for non-nullable field
func (u *User) GetAge() int32 {
	return u.age
}
