package transport

import (
	"mime"
	"net/http"
	"strings"
)

const (
	acceptApplicationJson                = "application/json"
	acceptApplicationGraphqlResponseJson = "application/graphql-response+json"
)

func determineResponseContentType(
	explicitHeaders map[string][]string,
	r *http.Request,
	useGrapQLResponseJsonByDefault bool,
) string {
	for k, v := range explicitHeaders {
		if strings.EqualFold(k, "Content-Type") {
			return v[0]
		}
	}

	accept := r.Header.Get("Accept")
	if accept == "" {
		if useGrapQLResponseJsonByDefault {
			return acceptApplicationGraphqlResponseJson
		}
		return acceptApplicationJson
	}

	for _, acceptPart := range strings.Split(accept, ",") {
		mediaType, _, err := mime.ParseMediaType(strings.TrimSpace(acceptPart))
		if err != nil {
			continue
		}
		switch mediaType {
		case "*/*", "application/*":
			if useGrapQLResponseJsonByDefault {
				return acceptApplicationGraphqlResponseJson
			}
			return acceptApplicationJson
		case "application/json":
			return acceptApplicationJson
		case "application/graphql-response+json":
			return acceptApplicationGraphqlResponseJson
		}
	}

	return acceptApplicationGraphqlResponseJson
}

func writeHeaders(w http.ResponseWriter, headers map[string][]string) {
	if len(headers) == 0 {
		headers = map[string][]string{
			// Stay with application/json (not application/graphql-response+json)
			// as it is not an actively supported protocol for now
			"Content-Type": {"application/json"},
		}
	}

	for key, values := range headers {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
}

func mergeHeaders(baseHeaders, additionalHeaders map[string][]string) map[string][]string {
	result := make(map[string][]string)
	for k, v := range baseHeaders {
		result[k] = v
	}
	for key, values := range additionalHeaders {
		result[key] = values
	}
	return result
}
