package scalar

import (
	"fmt"
	"io"
	"time"
)

// Time is a custom scalar type for time.Time
type Time time.Time

// MarshalGQL implements graphql.Marshaler interface
func (t Time) MarshalGQL(w io.Writer) {
	io.WriteString(w, fmt.Sprintf(`"%s"`, time.Time(t).Format(time.RFC3339)))
}

// UnmarshalGQL implements graphql.Unmarshaler interface
func (t *Time) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("time must be a string")
	}

	parsed, err := time.Parse(time.RFC3339, str)
	if err != nil {
		return err
	}

	*t = Time(parsed)
	return nil
}