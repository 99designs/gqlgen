package graphql

import (
	"encoding/json"
	"fmt"
	"io"
)

var (
	invalidUpload = "%T is not an Upload"
)

type Upload struct {
	File        io.ReadSeeker `json:"file"`
	Filename    string        `json:"filename"`
	Size        int64         `json:"size"`
	ContentType string        `json:"contentType"`
}

func MarshalUpload(f Upload) Marshaler {
	return WriterFunc(func(w io.Writer) {
		io.Copy(w, f.File)
	})
}

// UnmarshalUpload reads an Upload from a JSON-encoded value.
func UnmarshalUpload(v any) (Upload, error) {
	switch t := v.(type) {
	case Upload:
		return t, nil
	case map[string]interface{}:
		upload, err := unmarshalUploadMap(t)
		if err != nil {
			return Upload{}, fmt.Errorf(invalidUpload, v)
		}

		return upload, nil
	default:
		return Upload{}, fmt.Errorf(invalidUpload, v)
	}
}

// unmarshalUploadMap reads an Upload from a map value and returns the struct if it's not empty
func unmarshalUploadMap(m map[string]interface{}) (Upload, error) {
	var upload Upload

	out, err := json.Marshal(m)
	if err != nil {
		return upload, err
	}

	// Unmarshal the JSON-encoded value into the Upload struct
	// ignoring the error because we want to return the Upload struct if it's not empty
	json.Unmarshal(out, &upload) // nolint: errcheck

	if (Upload{}) == upload {
		return Upload{}, fmt.Errorf(invalidUpload, m)
	}

	return upload, nil
}
