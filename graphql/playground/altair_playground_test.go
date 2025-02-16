package playground

import (
	"net/http"
	"testing"
)

func TestAltairHandler_Integrity(t *testing.T) {
	testResourceIntegrity(t, func(title, endpoint string) http.HandlerFunc { return AltairHandler(title, endpoint, nil) })
}
