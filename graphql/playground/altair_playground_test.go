package playground

import (
	"testing"
)

func TestAltairHandler_Integrity(t *testing.T) {
	testResourceIntegrity(t, AltairHandler)
}
