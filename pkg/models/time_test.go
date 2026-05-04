package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestCustomTimeUnmarshalJSON(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		input     string
		wantZero  bool
		wantValue time.Time
		wantErr   bool
	}{
		{
			name:      "null",
			input:     `null`,
			wantZero:  true,
			wantValue: time.Time{},
		},
		{
			name:      "empty string",
			input:     `""`,
			wantZero:  true,
			wantValue: time.Time{},
		},
		{
			name:      "custom format",
			input:     `"2026-05-04 23:45:00"`,
			wantValue: time.Date(2026, 5, 4, 23, 45, 0, 0, time.UTC),
		},
		{
			name:      "rfc3339",
			input:     `"2026-05-04T23:45:00Z"`,
			wantValue: time.Date(2026, 5, 4, 23, 45, 0, 0, time.UTC),
		},
		{
			name:    "invalid",
			input:   `"not-a-time"`,
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var value CustomTime
			err := json.Unmarshal([]byte(tc.input), &value)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tc.wantZero {
				if !value.IsZero() {
					t.Fatalf("expected zero time, got %v", value.Time)
				}
				return
			}

			if !value.Time.Equal(tc.wantValue) {
				t.Fatalf("unexpected time: got %v want %v", value.Time, tc.wantValue)
			}
		})
	}
}

func TestCustomDateUnmarshalJSON(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		input     string
		wantZero  bool
		wantValue time.Time
		wantErr   bool
	}{
		{
			name:     "null",
			input:    `null`,
			wantZero: true,
		},
		{
			name:     "empty string",
			input:    `""`,
			wantZero: true,
		},
		{
			name:      "valid date",
			input:     `"2026-05-04"`,
			wantValue: time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC),
		},
		{
			name:    "invalid date",
			input:   `"2026/05/04"`,
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var value CustomDate
			err := json.Unmarshal([]byte(tc.input), &value)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tc.wantZero {
				if !value.IsZero() {
					t.Fatalf("expected zero date, got %v", value.Time)
				}
				return
			}

			if !value.Time.Equal(tc.wantValue) {
				t.Fatalf("unexpected date: got %v want %v", value.Time, tc.wantValue)
			}
		})
	}
}

func TestCustomTimeScan(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 5, 4, 23, 45, 0, 0, time.UTC)
	testCases := []struct {
		name      string
		input     interface{}
		wantZero  bool
		wantValue time.Time
		wantErr   bool
	}{
		{name: "nil", input: nil, wantZero: true},
		{name: "time", input: base, wantValue: base},
		{name: "string", input: "2026-05-04 23:45:00", wantValue: base},
		{name: "bytes", input: []byte("2026-05-04T23:45:00Z"), wantValue: base},
		{name: "empty string", input: "", wantZero: true},
		{name: "invalid", input: 123, wantErr: true},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var value CustomTime
			err := value.Scan(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tc.wantZero {
				if !value.IsZero() {
					t.Fatalf("expected zero time, got %v", value.Time)
				}
				return
			}

			if !value.Time.Equal(tc.wantValue) {
				t.Fatalf("unexpected time: got %v want %v", value.Time, tc.wantValue)
			}
		})
	}
}

func TestCustomDateScan(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC)
	testCases := []struct {
		name      string
		input     interface{}
		wantZero  bool
		wantValue time.Time
		wantErr   bool
	}{
		{name: "nil", input: nil, wantZero: true},
		{name: "time", input: base, wantValue: base},
		{name: "string", input: "2026-05-04", wantValue: base},
		{name: "bytes", input: []byte("2026-05-04"), wantValue: base},
		{name: "empty string", input: "", wantZero: true},
		{name: "invalid", input: 123, wantErr: true},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var value CustomDate
			err := value.Scan(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tc.wantZero {
				if !value.IsZero() {
					t.Fatalf("expected zero date, got %v", value.Time)
				}
				return
			}

			if !value.Time.Equal(tc.wantValue) {
				t.Fatalf("unexpected date: got %v want %v", value.Time, tc.wantValue)
			}
		})
	}
}
