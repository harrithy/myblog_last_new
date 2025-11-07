package models

import (
	"fmt"
	"time"
)

const (
	CustomTimeFormat = "2006-01-02 15:04:05"
	DateFormat       = "2006-01-02"
)

// CustomTime is a wrapper around time.Time to customize JSON marshalling
type CustomTime struct {
	time.Time
}

// CustomDate is a wrapper around time.Time for date-only formatting
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
	// Trim quotes
	str := string(data[1 : len(data)-1])

	// Try parsing multiple formats
	parsedTime, err := time.Parse(CustomTimeFormat, str)
	if err != nil {
		parsedTime, err = time.Parse(time.RFC3339, str)
		if err != nil {
			return err
		}
	}

	t.Time = parsedTime
	return nil
}

// UnmarshalJSON implements the json.Unmarshaler interface for CustomDate.
// The date is expected to be in "YYYY-MM-DD" format.
func (d *CustomDate) UnmarshalJSON(data []byte) error {
	// Trim quotes
	str := string(data[1 : len(data)-1])

	parsedTime, err := time.Parse(DateFormat, str)
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
	if vt, ok := value.(time.Time); ok {
		t.Time = vt
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
	if vt, ok := value.(time.Time); ok {
		d.Time = vt
		return nil
	}
	return fmt.Errorf("failed to scan CustomDate: %v", value)
}
