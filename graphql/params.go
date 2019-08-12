package graphql

import "net/http"

type ParsedParams struct {
	Query         string                 `json:"query"`
	OperationName string                 `json:"operationName"`
	Variables     map[string]interface{} `json:"variables"`
	Extensions    *Extensions            `json:"extensions"`
}

type Extensions struct {
	PersistedQuery *PersistedQuery `json:"persistedQuery"`
}

type PersistedQuery struct {
	Sha256  string `json:"sha256Hash"`
	Version int64  `json:"version"`
}

type ParserFunc func(http.ResponseWriter, *http.Request, ParsedParams)
type ParserHandlerFunc func(ParserFunc) http.HandlerFunc
