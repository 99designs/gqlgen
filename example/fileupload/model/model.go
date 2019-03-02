package model

import (
	"github.com/99designs/gqlgen/graphql"
	"github.com/pkg/errors"
	"io"
	"mime/multipart"
)

type Upload struct {
	File     multipart.File
	FileName string
	Size     int64
}

// if the type referenced in .gqlgen.yml is a function that returns a marshaller we can use it to encode and decode
// onto any existing go type.
func (u *Upload) UnmarshalGQL(v interface{}) error {
	data, ok := v.(graphql.Upload)
	if !ok {
		return errors.New("upload should be a graphql Upload")
	}
	*u = Upload{
		File:     data.File,
		FileName: data.Filename,
		Size:     data.Size,
	}
	return nil
}

func (u Upload) MarshalGQL(w io.Writer) {
	io.Copy(w, u.File)
}
