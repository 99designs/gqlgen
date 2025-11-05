package graphql

import (
	"bytes"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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

func TestMarshalTime(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		expected string
	}{
		{
			name:     "normal time with nanoseconds",
			input:    time.Date(2023, 10, 15, 14, 30, 45, 123456789, time.UTC),
			expected: `"2023-10-15T14:30:45.123456789Z"`,
		},
		{
			name:     "normal time without nanoseconds",
			input:    time.Date(2023, 10, 15, 14, 30, 45, 0, time.UTC),
			expected: `"2023-10-15T14:30:45Z"`,
		},
		{
			name:     "time with timezone offset",
			input:    time.Date(2023, 10, 15, 14, 30, 45, 0, time.FixedZone("EST", -5*60*60)),
			expected: `"2023-10-15T14:30:45-05:00"`,
		},
		{
			name:     "epoch time",
			input:    time.Unix(0, 0).UTC(),
			expected: `"1970-01-01T00:00:00Z"`,
		},
		{
			name:     "time at start of day",
			input:    time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			expected: `"2023-01-01T00:00:00Z"`,
		},
		{
			name:     "time at end of day",
			input:    time.Date(2023, 12, 31, 23, 59, 59, 999999999, time.UTC),
			expected: `"2023-12-31T23:59:59.999999999Z"`,
		},
		{
			name:     "leap year february 29",
			input:    time.Date(2024, 2, 29, 12, 0, 0, 0, time.UTC),
			expected: `"2024-02-29T12:00:00Z"`,
		},
		{
			name:     "far future date",
			input:    time.Date(2999, 12, 31, 23, 59, 59, 0, time.UTC),
			expected: `"2999-12-31T23:59:59Z"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			marshaler := MarshalTime(tt.input)
			marshaler.MarshalGQL(buf)

			assert.Equal(t, tt.expected, buf.String())
		})
	}

	t.Run("zero time returns null", func(t *testing.T) {
		zeroTime := time.Time{}
		marshaler := MarshalTime(zeroTime)

		assert.Equal(t, Null, marshaler, "zero time should return Null marshaler")
	})

	t.Run("zero time writes null to buffer", func(t *testing.T) {
		buf := &bytes.Buffer{}
		MarshalTime(time.Time{}).MarshalGQL(buf)

		assert.Equal(t, "null", buf.String())
	})
}

func TestUnmarshalTime(t *testing.T) {
	t.Run("RFC3339Nano format", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected time.Time
		}{
			{
				name:     "with nanoseconds",
				input:    "2023-10-15T14:30:45.123456789Z",
				expected: time.Date(2023, 10, 15, 14, 30, 45, 123456789, time.UTC),
			},
			{
				name:     "without nanoseconds",
				input:    "2023-10-15T14:30:45Z",
				expected: time.Date(2023, 10, 15, 14, 30, 45, 0, time.UTC),
			},
			{
				name:     "with milliseconds",
				input:    "2023-10-15T14:30:45.123Z",
				expected: time.Date(2023, 10, 15, 14, 30, 45, 123000000, time.UTC),
			},
			{
				name:     "with microseconds",
				input:    "2023-10-15T14:30:45.123456Z",
				expected: time.Date(2023, 10, 15, 14, 30, 45, 123456000, time.UTC),
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := UnmarshalTime(tt.input)
				require.NoError(t, err)
				assert.True(t, tt.expected.Equal(result), "expected %v, got %v", tt.expected, result)
			})
		}
	})

	t.Run("RFC3339 format", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected time.Time
		}{
			{
				name:     "with timezone offset positive",
				input:    "2023-10-15T14:30:45+05:00",
				expected: time.Date(2023, 10, 15, 14, 30, 45, 0, time.FixedZone("", 5*60*60)),
			},
			{
				name:     "with timezone offset negative",
				input:    "2023-10-15T14:30:45-08:00",
				expected: time.Date(2023, 10, 15, 14, 30, 45, 0, time.FixedZone("", -8*60*60)),
			},
			{
				name:     "UTC with Z suffix",
				input:    "2023-10-15T14:30:45Z",
				expected: time.Date(2023, 10, 15, 14, 30, 45, 0, time.UTC),
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := UnmarshalTime(tt.input)
				require.NoError(t, err)
				assert.True(t, tt.expected.Equal(result), "expected %v, got %v", tt.expected, result)
			})
		}
	})

	t.Run("DateTime format", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected time.Time
		}{
			{
				name:     "standard datetime",
				input:    "2023-10-15 14:30:45",
				expected: time.Date(2023, 10, 15, 14, 30, 45, 0, time.UTC),
			},
			{
				name:     "datetime with single digit day",
				input:    "2023-01-05 09:30:45",
				expected: time.Date(2023, 1, 5, 9, 30, 45, 0, time.UTC),
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := UnmarshalTime(tt.input)
				require.NoError(t, err)
				assert.True(t, tt.expected.Equal(result), "expected %v, got %v", tt.expected, result)
			})
		}
	})

	t.Run("null and empty values", func(t *testing.T) {
		tests := []struct {
			name     string
			input    any
			expected time.Time
			wantErr  bool
		}{
			{
				name:     "nil value",
				input:    nil,
				expected: time.Time{},
				wantErr:  false,
			},
			{
				name:     "empty string",
				input:    "",
				expected: time.Time{},
				wantErr:  false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := UnmarshalTime(tt.input)
				if tt.wantErr {
					assert.Error(t, err)
				} else {
					require.NoError(t, err)
					assert.True(t, tt.expected.IsZero())
					assert.True(t, result.IsZero())
				}
			})
		}
	})

	t.Run("error cases", func(t *testing.T) {
		tests := []struct {
			name    string
			input   any
			wantErr string
		}{
			{
				name:    "invalid format",
				input:   "not a time",
				wantErr: "time should be RFC3339Nano formatted string",
			},
			{
				name:    "wrong type - int",
				input:   12345,
				wantErr: "time should be RFC3339Nano formatted string",
			},
			{
				name:    "wrong type - bool",
				input:   true,
				wantErr: "time should be RFC3339Nano formatted string",
			},
			{
				name:    "wrong type - struct",
				input:   struct{ Value string }{Value: "test"},
				wantErr: "time should be RFC3339Nano formatted string",
			},
			{
				name:    "invalid date",
				input:   "2023-13-45T14:30:45Z",
				wantErr: "time should be RFC3339Nano formatted string",
			},
			{
				name:    "invalid time",
				input:   "2023-10-15T25:70:90Z",
				wantErr: "time should be RFC3339Nano formatted string",
			},
			{
				name:    "malformed RFC3339",
				input:   "2023-10-15 14:30:45Z",
				wantErr: "time should be RFC3339Nano formatted string",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := UnmarshalTime(tt.input)
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			})
		}
	})

	t.Run("round trip consistency", func(t *testing.T) {
		testTimes := []time.Time{
			time.Date(2023, 10, 15, 14, 30, 45, 123456789, time.UTC),
			time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2023, 12, 31, 23, 59, 59, 999999999, time.UTC),
			time.Unix(0, 0).UTC(),
			time.Now().UTC(),
		}

		for i, originalTime := range testTimes {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				// Marshal
				buf := &bytes.Buffer{}
				MarshalTime(originalTime).MarshalGQL(buf)

				// Unquote
				str, err := strconv.Unquote(buf.String())
				require.NoError(t, err)

				// Unmarshal
				parsedTime, err := UnmarshalTime(str)
				require.NoError(t, err)

				// Compare
				assert.True(t, originalTime.Equal(parsedTime),
					"round trip failed: original=%v, parsed=%v", originalTime, parsedTime)
			})
		}
	})
}

func TestMarshalTime_EdgeCases(t *testing.T) {
	t.Run("time with different locations same instant", func(t *testing.T) {
		utcTime := time.Date(2023, 10, 15, 14, 30, 45, 0, time.UTC)
		estTime := utcTime.In(time.FixedZone("EST", -5*60*60))

		utcBuf := &bytes.Buffer{}
		MarshalTime(utcTime).MarshalGQL(utcBuf)

		estBuf := &bytes.Buffer{}
		MarshalTime(estTime).MarshalGQL(estBuf)

		// Should produce different string representations
		assert.NotEqual(t, utcBuf.String(), estBuf.String())

		// But should unmarshal to the same instant
		utcStr, _ := strconv.Unquote(utcBuf.String())
		estStr, _ := strconv.Unquote(estBuf.String())

		utcParsed, err := UnmarshalTime(utcStr)
		require.NoError(t, err)
		estParsed, err := UnmarshalTime(estStr)
		require.NoError(t, err)

		assert.True(t, utcParsed.Equal(estParsed))
	})

	t.Run("precision preservation", func(t *testing.T) {
		// Test that nanosecond precision is preserved
		timeWithNanos := time.Date(2023, 10, 15, 14, 30, 45, 123456789, time.UTC)

		buf := &bytes.Buffer{}
		MarshalTime(timeWithNanos).MarshalGQL(buf)

		str, err := strconv.Unquote(buf.String())
		require.NoError(t, err)

		parsed, err := UnmarshalTime(str)
		require.NoError(t, err)

		assert.Equal(t, timeWithNanos.Nanosecond(), parsed.Nanosecond())
		assert.True(t, timeWithNanos.Equal(parsed))
	})
}

func TestMarshalDate(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		expected string
	}{
		{
			name:     "normal time with nanoseconds",
			input:    time.Date(2023, 10, 15, 14, 30, 45, 123456789, time.UTC),
			expected: `"2023-10-15"`,
		},
		{
			name:     "normal time without nanoseconds",
			input:    time.Date(2023, 10, 15, 14, 30, 45, 0, time.UTC),
			expected: `"2023-10-15"`,
		},
		{
			name:     "time with timezone offset",
			input:    time.Date(2023, 10, 15, 14, 30, 45, 0, time.FixedZone("EST", -5*60*60)),
			expected: `"2023-10-15"`,
		},
		{
			name:     "epoch time",
			input:    time.Unix(0, 0).UTC(),
			expected: `"1970-01-01"`,
		},
		{
			name:     "time at start of day",
			input:    time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			expected: `"2023-01-01"`,
		},
		{
			name:     "time at end of day",
			input:    time.Date(2023, 12, 31, 23, 59, 59, 999999999, time.UTC),
			expected: `"2023-12-31"`,
		},
		{
			name:     "leap year february 29",
			input:    time.Date(2024, 2, 29, 12, 0, 0, 0, time.UTC),
			expected: `"2024-02-29"`,
		},
		{
			name:     "far future date",
			input:    time.Date(2999, 12, 31, 23, 59, 59, 0, time.UTC),
			expected: `"2999-12-31"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			marshaler := MarshalDate(tt.input)
			marshaler.MarshalGQL(buf)

			assert.Equal(t, tt.expected, buf.String())
		})
	}

	t.Run("zero time returns null", func(t *testing.T) {
		zeroTime := time.Time{}
		marshaler := MarshalDate(zeroTime)

		assert.Equal(t, Null, marshaler, "zero time should return Null marshaler")
	})

	t.Run("zero time writes null to buffer", func(t *testing.T) {
		buf := &bytes.Buffer{}
		MarshalDate(time.Time{}).MarshalGQL(buf)

		assert.Equal(t, "null", buf.String())
	})
}

func TestUnmarshalDate(t *testing.T) {
	t.Run("DateOnly format", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected time.Time
		}{
			{
				name:     "standard date",
				input:    "2023-10-15",
				expected: time.Date(2023, 10, 15, 0, 0, 0, 0, time.UTC),
			},
			{
				name:     "january first",
				input:    "2023-01-01",
				expected: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			{
				name:     "december last",
				input:    "2023-12-31",
				expected: time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC),
			},
			{
				name:     "leap year february 29",
				input:    "2024-02-29",
				expected: time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC),
			},
			{
				name:     "single digit month and day",
				input:    "2023-01-05",
				expected: time.Date(2023, 1, 5, 0, 0, 0, 0, time.UTC),
			},
			{
				name:     "epoch date",
				input:    "1970-01-01",
				expected: time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := UnmarshalDate(tt.input)
				require.NoError(t, err)
				assert.True(t, tt.expected.Equal(result), "expected %v, got %v", tt.expected, result)
			})
		}
	})

	t.Run("null and empty values", func(t *testing.T) {
		tests := []struct {
			name     string
			input    any
			expected time.Time
			wantErr  bool
		}{
			{
				name:     "nil value",
				input:    nil,
				expected: time.Time{},
				wantErr:  false,
			},
			{
				name:     "empty string",
				input:    "",
				expected: time.Time{},
				wantErr:  false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := UnmarshalDate(tt.input)
				if tt.wantErr {
					assert.Error(t, err)
				} else {
					require.NoError(t, err)
					assert.True(t, tt.expected.IsZero())
					assert.True(t, result.IsZero())
				}
			})
		}
	})

	t.Run("error cases", func(t *testing.T) {
		tests := []struct {
			name    string
			input   any
			wantErr string
		}{
			{
				name:    "invalid format",
				input:   "not a date",
				wantErr: "DateOnly",
			},
			{
				name:    "wrong type - int",
				input:   12345,
				wantErr: "DateOnly formatted string",
			},
			{
				name:    "wrong type - bool",
				input:   true,
				wantErr: "DateOnly formatted string",
			},
			{
				name:    "wrong type - struct",
				input:   struct{ Value string }{Value: "test"},
				wantErr: "DateOnly formatted string",
			},
			{
				name:    "invalid date - month out of range",
				input:   "2023-13-15",
				wantErr: "time should be DateOnly formatted string",
			},
			{
				name:    "invalid date - day out of range",
				input:   "2023-10-45",
				wantErr: "time should be DateOnly formatted string",
			},
			{
				name:    "RFC3339 format not accepted",
				input:   "2023-10-15T14:30:45Z",
				wantErr: "time should be DateOnly formatted string",
			},
			{
				name:    "DateTime format not accepted",
				input:   "2023-10-15 14:30:45",
				wantErr: "time should be DateOnly formatted string",
			},
			{
				name:    "incomplete date",
				input:   "2023-10",
				wantErr: "time should be DateOnly formatted string",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := UnmarshalDate(tt.input)
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			})
		}
	})

	t.Run("round trip consistency", func(t *testing.T) {
		testTimes := []time.Time{
			time.Date(2023, 10, 15, 14, 30, 45, 123456789, time.UTC),
			time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2023, 12, 31, 23, 59, 59, 999999999, time.UTC),
			time.Unix(0, 0).UTC(),
			time.Now().UTC(),
		}

		for i, originalTime := range testTimes {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				// Marshal
				buf := &bytes.Buffer{}
				MarshalDate(originalTime).MarshalGQL(buf)

				// Unquote
				str, err := strconv.Unquote(buf.String())
				require.NoError(t, err)

				// Unmarshal
				parsedTime, err := UnmarshalDate(str)
				require.NoError(t, err)

				// Compare - dates should match (time components are dropped)
				expectedDate := time.Date(originalTime.Year(), originalTime.Month(), originalTime.Day(), 0, 0, 0, 0, time.UTC)
				assert.True(t, expectedDate.Equal(parsedTime),
					"round trip failed: expected=%v, parsed=%v", expectedDate, parsedTime)
			})
		}
	})
}

func TestMarshalDate_EdgeCases(t *testing.T) {
	t.Run("time components are ignored", func(t *testing.T) {
		time1 := time.Date(2023, 10, 15, 0, 0, 0, 0, time.UTC)
		time2 := time.Date(2023, 10, 15, 14, 30, 45, 123456789, time.UTC)

		buf1 := &bytes.Buffer{}
		MarshalDate(time1).MarshalGQL(buf1)

		buf2 := &bytes.Buffer{}
		MarshalDate(time2).MarshalGQL(buf2)

		// Both should produce the same date string
		assert.Equal(t, buf1.String(), buf2.String())
		assert.Equal(t, `"2023-10-15"`, buf1.String())
	})

	t.Run("timezone is preserved in format but only date is shown", func(t *testing.T) {
		utcTime := time.Date(2023, 10, 15, 14, 30, 45, 0, time.UTC)
		estTime := time.Date(2023, 10, 15, 14, 30, 45, 0, time.FixedZone("EST", -5*60*60))

		utcBuf := &bytes.Buffer{}
		MarshalDate(utcTime).MarshalGQL(utcBuf)

		estBuf := &bytes.Buffer{}
		MarshalDate(estTime).MarshalGQL(estBuf)

		// Both should produce date strings
		assert.Equal(t, `"2023-10-15"`, utcBuf.String())
		assert.Equal(t, `"2023-10-15"`, estBuf.String())
	})
}
