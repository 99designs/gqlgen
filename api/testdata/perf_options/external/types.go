// Package external contains types that are mapped via @goModel directive.
// This package is separate from the model package to test that gqlgen
// correctly handles type mappings with performance options enabled.
package external

// LocationInfo is a type that will be mapped via @goModel directive.
type LocationInfo struct {
	Country   string
	City      string
	Latitude  float64
	Longitude float64
}

// NewLocationInfo creates a new LocationInfo
func NewLocationInfo(country, city string, lat, lng float64) *LocationInfo {
	return &LocationInfo{
		Country:   country,
		City:      city,
		Latitude:  lat,
		Longitude: lng,
	}
}

