package testdata

import "math"

type Shape interface {
	Area() float64
}

type ShapeUnion interface {
	Area() float64
}

type Circle struct {
	Radius float64
}

func (c *Circle) Area() float64 {
	return c.Radius * math.Pi * math.Pi
}

type Rectangle struct {
	Length, Width float64
}

func (r *Rectangle) Area() float64 {
	return r.Length * r.Width
}
