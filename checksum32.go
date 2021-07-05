package flate

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Checksum32 is a lightweight wrapper around uint32 that is used for 32-bit
// checksums, such as CRC-32 or Adler-32.  It stringifies to hexadecimal
// format.
type Checksum32 uint32

// GoString returns the Go string representation of this Checksum32 value.
func (csum Checksum32) GoString() string {
	return fmt.Sprintf("Checksum32(%#08x)", uint32(csum))
}

// String returns the string representation of this Checksum32 value.
func (csum Checksum32) String() string {
	return fmt.Sprintf("%#08x", uint32(csum))
}

// MarshalJSON returns the JSON representation of this Checksum32 value.
func (csum Checksum32) MarshalJSON() ([]byte, error) {
	return json.Marshal(csum.String())
}

// UnmarshalJSON parses the JSON representation of a Checksum32 value.
func (csum *Checksum32) UnmarshalJSON(raw []byte) error {
	var str string
	if err := json.Unmarshal(raw, &str); err != nil {
		return err
	}
	str = strings.TrimPrefix(str, "0x")
	u64, err := strconv.ParseUint(str, 16, 32)
	if err != nil {
		return err
	}
	*csum = Checksum32(u64)
	return nil
}

var _ fmt.GoStringer = Checksum32(0)
var _ fmt.Stringer = Checksum32(0)
var _ json.Marshaler = Checksum32(0)
var _ json.Unmarshaler = (*Checksum32)(nil)
