package graphql

import (
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
