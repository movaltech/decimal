package decimal

import (
	"database/sql/driver"
	"fmt"
)

// MarshalJSON encodes d as a quoted decimal string, e.g. "\"10.50\"".
//
// Encoding as a JSON *string* rather than a bare number is deliberate: JSON
// numbers are conventionally decoded into float64 by consumers (including
// Go's own encoding/json into interface{}, and JavaScript's JSON.parse), so
// a bare 10.50 is one hop away from silent corruption the moment it leaves
// this package. A quoted string forces the receiver to consciously parse it
// as a decimal.
func (d Decimal) MarshalJSON() ([]byte, error) {
	s := d.String()
	buf := make([]byte, 0, len(s)+2)
	buf = append(buf, '"')
	buf = append(buf, s...)
	buf = append(buf, '"')
	return buf, nil
}

// UnmarshalJSON decodes a JSON string or bare JSON number into d, using
// Parse in both cases so the value is never routed through float64. Bare
// numbers are accepted for interoperability with producers that did not
// follow this package's own MarshalJSON convention, but the safe wire
// format remains a quoted string.
func (d *Decimal) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return ErrInvalidSyntax
	}
	s := string(data)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	if s == "null" {
		*d = Decimal{}
		return nil
	}
	v, err := Parse(s)
	if err != nil {
		return err
	}
	*d = v
	return nil
}

// MarshalText implements encoding.TextMarshaler, encoding d as the same
// plain decimal text as String() (e.g. "10.50"). Go's encoding/json prefers
// MarshalJSON over this when both are implemented, so this does not change
// JSON behavior; it exists for the other encoders that only know about
// encoding.TextMarshaler — YAML, TOML, and encoding/json's own map-key
// encoding — so a Decimal stays exact in those paths too, not just JSON.
func (d Decimal) MarshalText() ([]byte, error) {
	return []byte(d.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler using Parse, the
// counterpart to MarshalText. An empty input decodes to the zero value,
// mirroring UnmarshalJSON's treatment of "null".
func (d *Decimal) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*d = Decimal{}
		return nil
	}
	v, err := Parse(string(text))
	if err != nil {
		return err
	}
	*d = v
	return nil
}

// Value implements database/sql/driver.Valuer, storing d as its exact
// decimal text. Pair this with a NUMERIC/DECIMAL column, never FLOAT/DOUBLE,
// or the database itself becomes the precision leak.
func (d Decimal) Value() (driver.Value, error) {
	return d.String(), nil
}

// Scan implements database/sql.Scanner, accepting the string/[]byte/int64
// forms a driver may hand back for a NUMERIC column. A float64 source is
// also accepted for compatibility with drivers that misreport NUMERIC as
// float, but by that point the driver has already introduced whatever
// imprecision float64 carries — Scan cannot recover digits the driver lost
// upstream, which is one more reason to configure the column and driver to
// avoid float in the first place.
func (d *Decimal) Scan(src interface{}) error {
	switch v := src.(type) {
	case nil:
		*d = Decimal{}
		return nil
	case string:
		parsed, err := Parse(v)
		if err != nil {
			return err
		}
		*d = parsed
		return nil
	case []byte:
		parsed, err := Parse(string(v))
		if err != nil {
			return err
		}
		*d = parsed
		return nil
	case int64:
		*d = NewFromInt64(v)
		return nil
	case float64:
		parsed, err := Parse(fmt.Sprintf("%v", v))
		if err != nil {
			return err
		}
		*d = parsed
		return nil
	default:
		return fmt.Errorf("decimal: unsupported Scan source type %T", src)
	}
}
