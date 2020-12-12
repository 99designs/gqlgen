package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net/textproto"
	"os"
	"strings"
)

type fileFormDataMap struct {
	mapKey string
	file   *os.File
}

func findFiles(parentMapKey string, variables map[string]interface{}) []*fileFormDataMap {
	files := []*fileFormDataMap{}
	for key, value := range variables {
		if v, ok := value.(map[string]interface{}); ok {
			files = append(files, findFiles(parentMapKey+"."+key, v)...)
		} else if v, ok := value.([]*os.File); ok {
			for i, file := range v {
				files = append(files, &fileFormDataMap{
					mapKey: fmt.Sprintf(`%s.%s.%d`, parentMapKey, key, i),
					file:   file,
				})
			}
		} else if v, ok := value.(*os.File); ok {
			files = append(files, &fileFormDataMap{
				mapKey: parentMapKey + "." + key,
				file:   v,
			})
		}
	}

	return files
}

// WithFiles encodes the outgoing request body as multipart form data for file variables
func WithFiles() Option {
	return func(bd *Request) {
		bodyBuf := &bytes.Buffer{}
		bodyWriter := multipart.NewWriter(bodyBuf)

		//-b7955bd2e1d17b67ac157b9e9ddb6238888caefc6f3541920a1debad284d
		// Content-Disposition: form-data; name="operations"
		//
		// {"query":"mutation ($input: Input!) {}","variables":{"input":{"file":{}}}
		requestBody, _ := json.Marshal(bd)
		bodyWriter.WriteField("operations", string(requestBody))

		// --b7955bd2e1d17b67ac157b9e9ddb6238888caefc6f3541920a1debad284d
		// Content-Disposition: form-data; name="map"
		//
		// `{ "0":["variables.input.file"] }`
		// or
		// `{ "0":["variables.input.files.0"], "1":["variables.input.files.1"] }`
		mapData := ""
		filesData := findFiles("variables", bd.Variables)
		if len(filesData) > 0 {
			mapDataFiles := []string{}
			for i, fileData := range filesData {
				mapDataFiles = append(
					mapDataFiles,
					fmt.Sprintf(`"%d":["%s"]`, i, fileData.mapKey),
				)
			}
			mapData = `{` + strings.Join(mapDataFiles, ",") + `}`
		}
		bodyWriter.WriteField("map", mapData)

		// --b7955bd2e1d17b67ac157b9e9ddb6238888caefc6f3541920a1debad284d
		// Content-Disposition: form-data; name="0"; filename="tempFile"
		// Content-Type: application/octet-stream
		//
		for i, fileData := range filesData {
			h := make(textproto.MIMEHeader)
			h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%d"; filename="%s"`, i, fileData.file.Name()))
			h.Set("Content-Type", "application/octet-stream")
			ff, _ := bodyWriter.CreatePart(h)
			b, _ := ioutil.ReadFile(fileData.file.Name())
			ff.Write(b)
		}
		bodyWriter.Close()

		bd.HTTP.Body = ioutil.NopCloser(bodyBuf)
		bd.HTTP.Header.Set("Content-Type", bodyWriter.FormDataContentType())
	}
}
