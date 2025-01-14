package playground

import (
	"net/http"
	"testing"
)

func TestApolloSandboxHandler_Integrity(t *testing.T) {
	testResourceIntegrity(t, func(title, endpoint string, options map[string]any) http.HandlerFunc {
		return ApolloSandboxHandler(title, endpoint)
	})
}
