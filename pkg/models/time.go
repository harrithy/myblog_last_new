package models

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	// CustomTimeFormat is the default JSON and text format for timestamp fields.
	CustomTimeFormat = "2006-01-02 15:04:05"
	// DateFormat is the default JSON and text format for date-only fields.
	DateFormat = "2006-01-02"
)

// CustomTime wraps time.Time to customize JSON marshalling.
type CustomTime struct {
	time.Time
}

// CustomDate wraps time.Time for date-only formatting.
type CustomDate struct {
	time.Time
}

// MarshalJSON implements the json.Marshaler interface for CustomTime.
// The time is formatted as "YYYY-MM-DD HH:MM:SS".
func (t CustomTime) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte("null"), nil
	}
	formatted := fmt.Sprintf(`"%s"`, t.Format(CustomTimeFormat))
	return []byte(formatted), nil
}

// MarshalJSON implements the json.Marshaler interface for CustomDate.
// The date is formatted as "YYYY-MM-DD".
func (d CustomDate) MarshalJSON() ([]byte, error) {
	if d.IsZero() {
		return []byte("null"), nil
	}
	formatted := fmt.Sprintf(`"%s"`, d.Format(DateFormat))
	return []byte(formatted), nil
}

// UnmarshalJSON implements the json.Unmarshaler interface for CustomTime.
// The time is expected to be in "YYYY-MM-DD HH:MM:SS" or RFC3339 format.
func (t *CustomTime) UnmarshalJSON(data []byte) error {
	if isJSONNull(data) {
		t.Time = time.Time{}
		return nil
	}

	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	parsedTime, err := parseFlexibleTime(str)
	if err != nil {
		return err
	}
	t.Time = parsedTime
	return nil
}

// UnmarshalJSON implements the json.Unmarshaler interface for CustomDate.
// The date is expected to be in "YYYY-MM-DD" format.
func (d *CustomDate) UnmarshalJSON(data []byte) error {
	if isJSONNull(data) {
		d.Time = time.Time{}
		return nil
	}

	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	parsedTime, err := parseDate(str)
	if err != nil {
		return err
	}

	d.Time = parsedTime
	return nil
}

// Scan implements the sql.Scanner interface for CustomTime.
func (t *CustomTime) Scan(value interface{}) error {
	if value == nil {
		t.Time = time.Time{}
		return nil
	}

	switch vt := value.(type) {
	case time.Time:
		t.Time = vt
		return nil
	case []byte:
		parsedTime, err := parseFlexibleTime(string(vt))
		if err != nil {
			return fmt.Errorf("failed to scan CustomTime: %w", err)
		}
		t.Time = parsedTime
		return nil
	case string:
		parsedTime, err := parseFlexibleTime(vt)
		if err != nil {
			return fmt.Errorf("failed to scan CustomTime: %w", err)
		}
		t.Time = parsedTime
		return nil
	}

	return fmt.Errorf("failed to scan CustomTime: %v", value)
}

// Scan implements the sql.Scanner interface for CustomDate.
func (d *CustomDate) Scan(value interface{}) error {
	if value == nil {
		d.Time = time.Time{}
		return nil
	}

	switch vt := value.(type) {
	case time.Time:
		d.Time = vt
		return nil
	case []byte:
		parsedTime, err := parseDate(string(vt))
		if err != nil {
			return fmt.Errorf("failed to scan CustomDate: %w", err)
		}
		d.Time = parsedTime
		return nil
	case string:
		parsedTime, err := parseDate(vt)
		if err != nil {
			return fmt.Errorf("failed to scan CustomDate: %w", err)
		}
		d.Time = parsedTime
		return nil
	}

	return fmt.Errorf("failed to scan CustomDate: %v", value)
}

func isJSONNull(data []byte) bool {
	return strings.EqualFold(strings.TrimSpace(string(data)), "null")
}

func parseFlexibleTime(value string) (time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, nil
	}

	if parsedTime, err := time.Parse(CustomTimeFormat, trimmed); err == nil {
		return parsedTime, nil
	}

	if parsedTime, err := time.Parse(time.RFC3339, trimmed); err == nil {
		return parsedTime, nil
	}

	return time.Time{}, fmt.Errorf("unsupported time format %q", value)
}

func parseDate(value string) (time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, nil
	}

	parsedTime, err := time.Parse(DateFormat, trimmed)
	if err != nil {
		return time.Time{}, err
	}

	return parsedTime, nil
}
