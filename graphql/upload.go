package graphql

import (
	"fmt"
	"io"
)

type Upload struct {
	FileData []byte
	Filename string
	Size     int64
}

func MarshalUpload(f Upload) Marshaler {
	return WriterFunc(func(w io.Writer) {
		w.Write(f.FileData)
	})
}

func UnmarshalUpload(v interface{}) (Upload, error) {
	upload, ok := v.(Upload)
	if !ok {
		return Upload{}, fmt.Errorf("%T is not an Upload", v)
	}
	return upload, nil
}
