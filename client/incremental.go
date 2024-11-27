package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
)

type Incremental struct {
	Close func() error
	Next  func(response any) error
}

type IncrementalInitialResponse = struct {
	Data       any             `json:"data"`
	Label      string          `json:"label"`
	Path       []any           `json:"path"`
	HasNext    bool            `json:"hasNext"`
	Errors     json.RawMessage `json:"errors"`
	Extensions map[string]any  `json:"extensions"`
}

type IncrementalPendingData struct {
	ID    string `json:"id"`
	Path  []any  `json:"path"`
	Label string `json:"label"`
}

type IncrementalCompletedData struct {
	ID string `json:"id"`
}

type IncrementalDataResponse struct {
	// ID         string          `json:"id"`
	// Items      []any           `json:"items"`
	Data       any             `json:"data"`
	Label      string          `json:"label"`
	Path       []any           `json:"path"`
	HasNext    bool            `json:"hasNext"`
	Errors     json.RawMessage `json:"errors"`
	Extensions map[string]any  `json:"extensions"`
}

type IncrementalResponse struct {
	// Pending     []IncrementalPendingData   `json:"pending"`
	// Completed   []IncrementalCompletedData `json:"completed"`
	Incremental []IncrementalDataResponse `json:"incremental"`
	HasNext     bool                      `json:"hasNext"`
	Errors      json.RawMessage           `json:"errors"`
	Extensions  map[string]any            `json:"extensions"`
}

func errorIncremental(err error) *Incremental {
	return &Incremental{
		Close: func() error { return nil },
		Next: func(response any) error {
			return err
		},
	}
}

func (p *Client) Incremental(ctx context.Context, query string, options ...Option) *Incremental {
	r, err := p.newRequest(query, options...)
	if err != nil {
		return errorIncremental(fmt.Errorf("request: %w", err))
	}
	r.Header.Set("Accept", "multipart/mixed")

	w := httptest.NewRecorder()
	p.h.ServeHTTP(w, r)

	res := w.Result()
	if res.StatusCode >= http.StatusBadRequest {
		return errorIncremental(fmt.Errorf("http %d: %s", w.Code, w.Body.String()))
	}
	mediaType, params, err := mime.ParseMediaType(res.Header.Get("Content-Type"))
	if err != nil {
		return errorIncremental(fmt.Errorf("parse content-type: %w", err))
	}
	if mediaType != "multipart/mixed" {
		return errorIncremental(fmt.Errorf("expected content-type multipart/mixed, got %s", mediaType))
	}
	boundary, ok := params["boundary"]
	if !ok || boundary == "" {
		return errorIncremental(fmt.Errorf("expected boundary in content-type"))
	}
	deferSpec, ok := params["deferspec"]
	if !ok || deferSpec == "" {
		return errorIncremental(fmt.Errorf("expected deferSpec in content-type"))
	}

	errCh := make(chan error, 1)
	initCh := make(chan IncrementalInitialResponse)
	nextCh := make(chan IncrementalResponse)

	ctx, cancel := context.WithCancel(ctx)
	go func() {
		defer cancel()
		defer res.Body.Close()

		initialResponse := true
		mr := multipart.NewReader(res.Body, boundary)
		for {
			type nextPart struct {
				*multipart.Part
				Err error
			}

			nextPartCh := make(chan nextPart)
			go func() {
				var next nextPart
				next.Part, next.Err = mr.NextPart()
				nextPartCh <- next
			}()

			var next nextPart
			select {
			case <-ctx.Done():
				if err := ctx.Err(); err != nil {
					errCh <- fmt.Errorf("context: %w", err)
				}
				return
			case next = <-nextPartCh:
			}

			if next.Err == io.EOF {
				break
			}
			if next.Err != nil {
				errCh <- fmt.Errorf("next part: %w", next.Err)
				return
			}
			if ct := next.Header.Get("Content-Type"); ct != "application/json" {
				errCh <- fmt.Errorf(`expected content-type "application/json", got %q`, ct)
				return
			}

			if initialResponse {
				initialResponse = false
				var data IncrementalInitialResponse
				if err = json.NewDecoder(next.Part).Decode(&data); err != nil {
					errCh <- fmt.Errorf("decode part: %w", err)
					return
				}
				initCh <- data
				close(initCh)
			} else {
				var data IncrementalResponse
				if err = json.NewDecoder(next.Part).Decode(&data); err != nil {
					errCh <- fmt.Errorf("decode part: %w", err)
					return
				}
				nextCh <- data
			}
		}
	}()

	return &Incremental{
		Close: func() error {
			cancel()
			return nil
		},
		Next: func(response any) error {
			var data any
			var rawErrors json.RawMessage

			select {
			case nextErr := <-errCh:
				return nextErr
			case initData, ok := <-initCh:
				if !ok {
					select {
					case nextErr := <-errCh:
						return nextErr
					case nextData := <-nextCh:
						data = nextData
						rawErrors = nextData.Errors
					}
				} else {
					data = initData
					rawErrors = initData.Errors
				}
			}
			// we want to unpack even if there is an error, so we can see partial responses
			unpackErr := unpack(data, response, p.dc)
			if rawErrors != nil {
				return RawJsonError{rawErrors}
			}
			return unpackErr
		},
	}
}
