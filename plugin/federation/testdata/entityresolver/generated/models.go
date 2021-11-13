// Code generated by github.com/99designs/gqlgen, DO NOT EDIT.

package generated

type Hello struct {
	Name      string `json:"name"`
	Secondary string `json:"secondary"`
}

func (Hello) IsEntity() {}

type HelloWithErrors struct {
	Name string `json:"name"`
}

func (HelloWithErrors) IsEntity() {}

type MultiHello struct {
	Name string `json:"name"`
}

func (MultiHello) IsEntity() {}

type MultiHelloByNamesInput struct {
	Name string `json:"Name"`
}

type MultiHelloWithError struct {
	Name string `json:"name"`
}

func (MultiHelloWithError) IsEntity() {}

type MultiHelloWithErrorByNamesInput struct {
	Name string `json:"Name"`
}

type PlanetRequires struct {
	Name     string `json:"name"`
	Size     int    `json:"size"`
	Diameter int    `json:"diameter"`
}

func (PlanetRequires) IsEntity() {}

type World struct {
	Foo   string `json:"foo"`
	Bar   int    `json:"bar"`
	Hello *Hello `json:"hello"`
}

func (World) IsEntity() {}

type WorldName struct {
	Name string `json:"name"`
}

func (WorldName) IsEntity() {}
