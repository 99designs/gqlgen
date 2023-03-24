package transport

import "net/http"

func writeHeaders(w http.ResponseWriter, headers map[string][]string) {
	if len(headers) == 0 {
		headers = map[string][]string{
			"Content-Type": {"application/json"},
		}
	}

	for key, values := range headers {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
}
