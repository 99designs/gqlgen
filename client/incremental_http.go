package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
)

type IncrementalHandler struct {
	close func() error
	next  func(response any) error
}

func (i *IncrementalHandler) Close() error {
	return i.close()
}

func (i *IncrementalHandler) Next(response any) error {
	return i.next(response)
}

type IncrementalInitialResponse struct {
	Data       any             `json:"data"`
	Label      string          `json:"label"`
	Path       []any           `json:"path"`
	HasNext    bool            `json:"hasNext"`
	Errors     json.RawMessage `json:"errors"`
	Extensions map[string]any  `json:"extensions"`
}

type IncrementalData struct {
	// Support for "items" for @stream is not yet available, only "data" for
	// @defer, as per the 2023 spec. Similarly, this retains a more complete
	// list of fields, but not "id," and represents a mid-point between the
	// 2022 and 2023 specs.

	Data       any             `json:"data"`
	Label      string          `json:"label"`
	Path       []any           `json:"path"`
	HasNext    bool            `json:"hasNext"`
	Errors     json.RawMessage `json:"errors"`
	Extensions map[string]any  `json:"extensions"`
}

type IncrementalResponse struct {
	// Does not include the pending or completed fields from the 2023 spec.

	Incremental []IncrementalData `json:"incremental"`
	HasNext     bool              `json:"hasNext"`
	Errors      json.RawMessage   `json:"errors"`
	Extensions  map[string]any    `json:"extensions"`
}

func errorIncremental(err error) *IncrementalHandler {
	return &IncrementalHandler{
		close: func() error { return nil },
		next: func(response any) error {
			return err
		},
	}
}

// IncrementalHTTP returns a GraphQL response handler for the current
// GQLGen implementation of the [incremental delivery over HTTP spec].
// The IncrementalHTTP spec provides for "streaming" responses triggered by
// the use of @stream or @defer as an alternate approach to SSE. To that end,
// the client retains the interface of the handler returned from
// Client.SSE.
//
// IncrementalHTTP delivery using multipart/mixed is just the structure
// of the response: the payloads are specified by the defer-stream spec,
// which are in transition. For more detail, see the links in the
// definition for transport.MultipartMixed. We use the name
// IncrementalHTTP here to distinguish from the multipart form upload
// (the term "multipart" usually referring to the latter).
//
// IncrementalHandler is not safe for concurrent use, or for production
// use at all.
//
// [incremental delivery over HTTP spec]: https://github.com/graphql/graphql-over-http/blob/main/rfcs/IncrementalDelivery.md
func (p *Client) IncrementalHTTP(ctx context.Context, query string, options ...Option) *IncrementalHandler {
	r, err := p.newRequest(query, options...)
	if err != nil {
		return errorIncremental(fmt.Errorf("request: %w", err))
	}
	r.Header.Set("Accept", "multipart/mixed")

	w := httptest.NewRecorder()
	p.h.ServeHTTP(w, r)

	// Remains open since we are reading from it incrementally.
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

	// TODO: worth checking the deferSpec either to confirm this client
	// supports it exactly, or simply to make sure it is within some
	// expected range.
	deferSpec, ok := params["deferspec"]
	if !ok || deferSpec == "" {
		return errorIncremental(errors.New("expected deferSpec in content-type"))
	}

	boundary, ok := params["boundary"]
	if !ok || boundary == "" {
		return errorIncremental(errors.New("expected boundary in content-type"))
	}
	mr := multipart.NewReader(res.Body, boundary)

	ctx, cancel := context.WithCancelCause(ctx)
	initial := true

	return &IncrementalHandler{
		close: func() error {
			res.Body.Close()
			cancel(context.Canceled)
			return nil
		},
		next: func(response any) (err error) {
			defer func() {
				if err != nil {
					res.Body.Close()
					cancel(err)
				}
			}()

			var data any
			var rawErrors json.RawMessage

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
				return ctx.Err()
			case next = <-nextPartCh:
			}

			if next.Err == io.EOF {
				res.Body.Close()
				cancel(context.Canceled)
				return nil
			}
			if err = next.Err; err != nil {
				return err
			}
			if ct := next.Header.Get("Content-Type"); ct != "application/json" {
				err = fmt.Errorf(`expected content-type "application/json", got %q`, ct)
				return err
			}

			if initial {
				initial = false
				data = IncrementalInitialResponse{}
			} else {
				data = IncrementalResponse{}
			}
			if err = json.NewDecoder(next.Part).Decode(&data); err != nil {
				return err
			}

			// We want to unpack even if there is an error, so we can see partial
			// responses.
			err = unpack(data, response, p.dc)
			if len(rawErrors) != 0 {
				err = RawJsonError{rawErrors}
				return err
			}
			return err
		},
	}
}
