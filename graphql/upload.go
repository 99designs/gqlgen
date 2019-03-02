package graphql

import (
	"mime/multipart"
)

type Upload struct {
	File     multipart.File
	Filename string
	Size     int64
}
