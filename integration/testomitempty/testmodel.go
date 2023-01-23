package testomitempty

type (
	NamedString  string
	NamedInt     int
	NamedInt8    int8
	NamedInt16   int16
	NamedInt32   int32
	NamedInt64   int64
	NamedBool    bool
	NamedFloat32 float32
	NamedFloat64 float64
	NamedUint    uint
	NamedUint8   uint8
	NamedUint16  uint16
	NamedUint32  uint32
	NamedUint64  uint64
	NamedID      int
)

type RemoteModelWithOmitempty struct {
	Description string `json:"newDesc,omitempty"`
}
