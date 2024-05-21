package graphql

import (
	"bytes"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTime(t *testing.T) {
	t.Run("symmetry", func(t *testing.T) {
		initialTime := time.Now()
		buf := bytes.NewBuffer([]byte{})
		MarshalTime(initialTime).MarshalGQL(buf)

		str, err := strconv.Unquote(buf.String())
		require.NoError(t, err)
		newTime, err := UnmarshalTime(str)
		require.NoError(t, err)

		require.True(t, initialTime.Equal(newTime), "expected times %v and %v to equal", initialTime, newTime)
	})
}

func TestUnmarshalTime(t *testing.T) {
	tests := []struct {
		name    string
		in      interface{}
		want    time.Time
		wantErr bool
	}{
		{
			name:    "RFC3339Nano",
			in:      "2022-02-10T10:20:30.123456789Z",
			want:    time.Date(2022, 02, 10, 10, 20, 30, 123456789, time.UTC),
			wantErr: false,
		},
		{
			name:    "RFC3339",
			in:      "2022-02-10T10:20:30Z",
			want:    time.Date(2022, 02, 10, 10, 20, 30, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "UTC ISO with time",
			in:      "2022-02-10 10:20:30",
			want:    time.Date(2022, 02, 10, 10, 20, 30, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "UTC ISO with time and nsec",
			in:      "2022-02-10 10:20:30.123456789",
			want:    time.Date(2022, 02, 10, 10, 20, 30, 123456789, time.UTC),
			wantErr: false,
		},
		{
			name:    "UTC ISO date",
			in:      "2022-02-10",
			want:    time.Date(2022, 02, 10, 0, 0, 0, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "Invalid format 1",
			in:      "20220210",
			want:    time.Time{},
			wantErr: true,
		},
		{
			name:    "Invalid format 1",
			in:      "2022-02-33",
			want:    time.Time{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := UnmarshalTime(tt.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalTime() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !got.Equal(tt.want) {
				t.Errorf("UnmarshalTime() = %v, want %v", got, tt.want)
			}
		})
	}
}
