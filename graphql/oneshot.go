package graphql

import "context"

// OneShot wraps a single Response into a ResponseHandler iterator.
// This is critical when short-circuiting a request to return an error from an OperationInterceptor.
// Without OneShot, streaming transports (e.g. WebSockets, Server-Sent Events) will loop infinitely
// because they expect the ResponseHandler to eventually return nil to indicate the end of the stream.
func OneShot(resp *Response) ResponseHandler {
	var oneshot bool

	return func(context context.Context) *Response {
		if oneshot {
			return nil
		}
		oneshot = true

		return resp
	}
}
