package external

type (
	ObjectID     int
	Manufacturer string // remote named string
	Count        uint32 // remote named uint32
)

const (
	ManufacturerTesla  Manufacturer = "TESLA"
	ManufacturerHonda  Manufacturer = "HONDA"
	ManufacturerToyota Manufacturer = "TOYOTA"
)
