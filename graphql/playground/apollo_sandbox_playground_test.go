package playground

import (
	"net/http"
	"testing"
)

func TestApolloSandboxHandler_Integrity(t *testing.T) {
	testResourceIntegrity(t, func(title, endpoint string) http.HandlerFunc {
		return ApolloSandboxHandler(title, endpoint)
	})
}
