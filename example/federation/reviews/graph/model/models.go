package model

type Product struct {
	ID           string        `json:"id"`
	Manufacturer *Manufacturer `json:"manufacturer"`
	Reviews      []*Review     `json:"reviews"`
}

func (Product) IsEntity() {}

type Review struct {
	Body    string
	Author  *User
	Product *Product
}

type User struct {
	ID    string     `json:"id"`
	Host  *EmailHost `json:"host"`
	Email string     `json:"email"`
	// Reviews []*Review  `json:"reviews"`
}

func (User) IsEntity() {}
