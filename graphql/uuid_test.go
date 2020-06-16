package graphql

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestUUID(t *testing.T) {
	generateUUID := func() uuid.UUID {
		u, err := uuid.NewRandom()
		assert.NoError(t, err)
		return u
	}

	uuidTD := []struct {
		uuid uuid.UUID
	}{
		{uuid: generateUUID()},
		{uuid: generateUUID()},
		{uuid: generateUUID()},
		{uuid: generateUUID()},
		{uuid: generateUUID()},
	}

	for _, v := range uuidTD {
		assert.Equal(t, strconv.Quote(v.uuid.String()), m2s(MarshalUUID(v.uuid)))
	}
}

func TestUnmarshalUUID(t *testing.T) {
	uuidTD := []struct {
		isGoodCase bool
		uuid       interface{}
	}{
		{isGoodCase: true, uuid: "12345678-1234-1234-1234-123456789012"},
		{isGoodCase: true, uuid: "{12345678-1234-1234-1234-123456789012}"},
		{isGoodCase: true, uuid: "urn:uuid:12345678-1234-1234-1234-123456789012"},
		{isGoodCase: true, uuid: "12345678901234567890123456789012"},
		{isGoodCase: false, uuid: ""},
		{isGoodCase: false, uuid: "1234567890123456789012"},
		{isGoodCase: false, uuid: "z2345678901234567890123456789012"},
		{isGoodCase: false, uuid: 42},
	}

	for i, v := range uuidTD {
		u, err := UnmarshalUUID(v.uuid)
		assert.NotNil(t, u)
		if v.isGoodCase {
			assert.NoError(t, err, fmt.Sprintf("case #%d", i))
		} else {
			assert.Error(t, err, fmt.Sprintf("case #%d", i))
		}
	}
}
