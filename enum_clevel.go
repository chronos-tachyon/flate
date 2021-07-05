package flate

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// CompressLevel indicates the desired effort / CPU time to expend in finding
// an optimal compression stream.
type CompressLevel int8

const (
	// DefaultCompression requests that the default value for CompressLevel
	// be selected.  This is currently equivalent to 6.
	DefaultCompression CompressLevel = -1

	// NoCompression requests that the data be stored literally, rather
	// than compressed.
	NoCompression CompressLevel = 0

	// FastestCompression requests that the data be compressed with the
	// greatest speed and least effort.
	FastestCompression CompressLevel = 1

	// BestCompression requests that the data be compressed with the
	// greatest effort and least speed.
	BestCompression CompressLevel = 9
)

// IsValid returns true if clevel is a valid CompressLevel constant.
func (clevel CompressLevel) IsValid() bool {
	return clevel >= DefaultCompression && clevel <= BestCompression
}

// GoString returns the Go string representation of this CompressLevel constant.
func (clevel CompressLevel) GoString() string {
	if clevel < 0 {
		return "DefaultCompression"
	}
	return fmt.Sprintf("CompressLevel(%d)", int(clevel))
}

// String returns the string representation of this CompressLevel constant.
func (clevel CompressLevel) String() string {
	if clevel < 0 {
		return strDefault
	}
	if clevel == 0 {
		return "none"
	}
	return fmt.Sprintf("%d", int(clevel))
}

// MarshalJSON returns the JSON representation of this CompressLevel constant.
func (clevel CompressLevel) MarshalJSON() ([]byte, error) {
	return json.Marshal(int(clevel))
}

// Parse parses a string representation of a CompressLevel constant.
func (clevel *CompressLevel) Parse(str string) error {
	if strings.EqualFold(str, strDefault) {
		*clevel = DefaultCompression
		return nil
	}

	if strings.EqualFold(str, "none") {
		*clevel = NoCompression
		return nil
	}

	u64, err := strconv.ParseUint(str, 10, 8)
	if err != nil {
		*clevel = DefaultCompression
		return err
	}
	if u64 > 9 {
		*clevel = DefaultCompression
		return fmt.Errorf("value %d is out of range", u64)
	}
	*clevel = CompressLevel(u64)
	return nil
}

var _ fmt.GoStringer = CompressLevel(0)
var _ fmt.Stringer = CompressLevel(0)
