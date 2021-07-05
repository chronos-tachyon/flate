package flate

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// WindowBits indicates the amount of RAM for the LZ77 sliding window to use.
type WindowBits byte

const (
	// DefaultWindowBits requests that the default value for WindowBits be
	// selected.  This is equivalent to 15 (MaxWindowBits).
	DefaultWindowBits WindowBits = 0

	// MinWindowBits is the smallest possible WindowBits.  As WindowBits
	// increases from 8 (MinWindowBits) to 15 (MaxWindowBits), the amount
	// of memory used for the sliding window doubles with each increase.  A
	// larger sliding window typically yields greater compression.
	MinWindowBits WindowBits = 8

	// MaxWindowBits is the largest possible WindowBits.  As WindowBits
	// increases from 8 (MinWindowBits) to 15 (MaxWindowBits), the amount
	// of memory used for the sliding window doubles with each increase.  A
	// larger sliding window typically yields greater compression.
	MaxWindowBits WindowBits = 15
)

// IsValid returns true if wbits is a valid WindowBits constant.
func (wbits WindowBits) IsValid() bool {
	return wbits == DefaultWindowBits || (wbits >= MinWindowBits && wbits <= MaxWindowBits)
}

// GoString returns the Go string representation of this WindowBits constant.
func (wbits WindowBits) GoString() string {
	if wbits < MinWindowBits {
		return "DefaultWindowBits"
	}
	return fmt.Sprintf("WindowBits(%d)", uint(wbits))
}

// String returns the string representation of this WindowBits constant.
func (wbits WindowBits) String() string {
	if wbits < MinWindowBits {
		return strDefault
	}
	return fmt.Sprintf("%d", uint(wbits))
}

// MarshalJSON returns the JSON representation of this WindowBits constant.
func (wbits WindowBits) MarshalJSON() ([]byte, error) {
	return json.Marshal(uint(wbits))
}

// Parse parses a string representation of a WindowBits constant.
func (wbits *WindowBits) Parse(str string) error {
	if strings.EqualFold(str, strDefault) {
		*wbits = DefaultWindowBits
		return nil
	}

	u64, err := strconv.ParseUint(str, 10, 8)
	if err != nil {
		*wbits = DefaultWindowBits
		return err
	}
	if u64 < uint64(MinWindowBits) {
		*wbits = DefaultWindowBits
		return fmt.Errorf("value %d is less than minimum %d", u64, uint64(MinWindowBits))
	}
	if u64 > uint64(MaxWindowBits) {
		*wbits = DefaultWindowBits
		return fmt.Errorf("value %d is less than maximum %d", u64, uint64(MaxWindowBits))
	}
	*wbits = WindowBits(u64)
	return nil
}

var _ fmt.GoStringer = WindowBits(0)
var _ fmt.Stringer = WindowBits(0)
