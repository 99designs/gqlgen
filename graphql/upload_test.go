package graphql

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmarshalUpload(t *testing.T) {
	// Create a ReadSeeker with nil value to test the Upload struct
	file := io.ReadSeeker(nil)

	tests := []struct {
		name    string
		input   any
		want    Upload
		wantErr bool
	}{
		{
			name: "valid Upload struct",
			input: Upload{
				File:        file,
				Filename:    "test.txt",
				Size:        1234,
				ContentType: "text/plain",
			},
			want: Upload{
				File:        file,
				Filename:    "test.txt",
				Size:        1234,
				ContentType: "text/plain",
			},
			wantErr: false,
		},
		{
			name:    "invalid type",
			input:   "invalid",
			want:    Upload{},
			wantErr: true,
		},
		{
			name: "valid JSON",
			input: map[string]interface{}{
				"file":        file,
				"filename":    "test.txt",
				"size":        1234,
				"contentType": "text/plain",
			},
			want: Upload{
				File:        file,
				Filename:    "test.txt",
				Size:        1234,
				ContentType: "text/plain",
			},
			wantErr: false,
		},
		{
			name: "invalid JSON",
			input: map[string]interface{}{
				"hello": "invalid",
			},
			want:    Upload{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := UnmarshalUpload(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}
