//go:generate ggraphqlc -out gen/generated.go

package readme

type Todo struct {
	ID     string
	Text   string
	Done   bool
	UserID int
}

type User struct {
	ID   string
	Name string
}
