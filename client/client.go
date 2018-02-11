// client is used internally for testing. See readme for alternatives
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/mitchellh/mapstructure"
	"github.com/vektah/gqlgen/neelance/errors"
)

// Client for graphql requests
type Client struct {
	url    string
	client *http.Client
}

// New creates a graphql client
func New(url string, client ...*http.Client) *Client {
	p := &Client{
		url: url,
	}

	if len(client) > 0 {
		p.client = client[0]
	} else {
		p.client = http.DefaultClient
	}
	return p
}

type Request struct {
	Query         string                 `json:"query"`
	Variables     map[string]interface{} `json:"variables,omitempty"`
	OperationName string                 `json:"operationName,omitempty"`
}

type Option func(r *Request)

func Var(name string, value interface{}) Option {
	return func(r *Request) {
		if r.Variables == nil {
			r.Variables = map[string]interface{}{}
		}

		r.Variables[name] = value
	}
}

func Operation(name string) Option {
	return func(r *Request) {
		r.OperationName = name
	}
}

func (p *Client) MustPost(query string, response interface{}, options ...Option) {
	if err := p.Post(query, response, options...); err != nil {
		panic(err)
	}
}

func (p *Client) Post(query string, response interface{}, options ...Option) error {
	r := Request{
		Query: query,
	}

	for _, option := range options {
		option(&r)
	}

	requestBody, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("encode: %s", err.Error())
	}

	rawResponse, err := p.client.Post(p.url, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return fmt.Errorf("post: %s", err.Error())
	}
	defer func() {
		_ = rawResponse.Body.Close()
	}()

	if rawResponse.StatusCode >= http.StatusBadRequest {
		responseBody, _ := ioutil.ReadAll(rawResponse.Body)
		return fmt.Errorf("http %d: %s", rawResponse.StatusCode, responseBody)
	}

	responseBody, err := ioutil.ReadAll(rawResponse.Body)
	if err != nil {
		return fmt.Errorf("read: %s", err.Error())
	}

	// decode it into map string first, let mapstructure do the final decode
	// because it can be much stricter about unknown fields.
	respDataRaw := map[string]interface{}{}
	err = json.Unmarshal(responseBody, &respDataRaw)
	if err != nil {
		return fmt.Errorf("decode: %s", err.Error())
	}

	respData := struct {
		Data   interface{}          `json:"data"`
		Errors []*errors.QueryError `json:"errors"`
	}{
		Data: response,
	}

	d, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:      &respData,
		TagName:     "json",
		ErrorUnused: true,
	})
	if err != nil {
		return fmt.Errorf("mapstructure: %s", err.Error())
	}

	err = d.Decode(respDataRaw)
	if err != nil {
		return fmt.Errorf("mapping: %s", err.Error())
	}

	if len(respData.Errors) > 0 {
		return fmt.Errorf("errors: %s", respData.Errors)
	}

	return nil
}
