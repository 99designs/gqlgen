package testserver

type IfaceBar struct {
	ID string `json:"id"`
	Y  string `json:"y"`
}

func (IfaceBar) isEntity() {}

type IfaceEntity interface {
	isEntity() // nolint: megacheck
}

type IfaceFoo struct {
	ID string `json:"id"`
	X  string `json:"x"`
}

func (IfaceFoo) isEntity() {}
