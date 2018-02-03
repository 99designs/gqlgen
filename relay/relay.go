package relay

import (
	"encoding/json"
	"net/http"

	"github.com/vektah/graphql-go/exec"
	"github.com/vektah/graphql-go/schema"
)

type Handler struct {
	Schema *schema.Schema
	Root   exec.Root
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var params struct {
		Query         string                 `json:"query"`
		OperationName string                 `json:"operationName"`
		Variables     map[string]interface{} `json:"variables"`
	}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response := exec.ExecuteRequest(h.Root, h.Schema, params.Query, params.OperationName, params.Variables)
	responseJSON, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseJSON)
}
