package playground

import (
	"testing"
)

func TestApolloSandboxHandler_Integrity(t *testing.T) {
	testResourceIntegrity(t, ApolloSandboxHandler)
}
