package serial

import "encoding/json"

type Serialization interface {
	Marshal(v interface{}) ([]byte, error)
	Unmarshal(data []byte, v interface{}) error
}

type defaultSerialization struct{}

func (d *defaultSerialization) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (d *defaultSerialization) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func Default() Serialization {
	return &defaultSerialization{}
}
