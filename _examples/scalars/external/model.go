package external

type (
	ObjectID     int
	Manufacturer string // remote named string
	Count        uint8  // remote named uint8
)

const (
	ManufacturerTesla  Manufacturer = "TESLA"
	ManufacturerHonda  Manufacturer = "HONDA"
	ManufacturerToyota Manufacturer = "TOYOTA"
)
