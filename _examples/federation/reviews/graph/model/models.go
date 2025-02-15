package model

type Product struct {
	ID           string        `json:"id"`
	Manufacturer *Manufacturer `json:"manufacturer"`
	Reviews      []*Review     `json:"reviews"`
}

func (Product) IsEntity()       {}
func (Product) IsThing()        {}
func (p Product) GetID() string { return p.ID }

type Review struct {
	Body        string
	Author      *User
	Product     *Product
	HostIDEmail string
}

type User struct {
	ID    string     `json:"id"`
	Host  *EmailHost `json:"host"`
	Email string     `json:"email"`
}

func (User) IsEntity() {}
