package flate

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// MemoryLevel indicates the amount of RAM for the Reader or Writer to use.
type MemoryLevel byte

const (
	// DefaultMemory requests that the default value for MemoryLevel be
	// selected.  This is currently equivalent to 9 (FastestMemory).
	DefaultMemory MemoryLevel = 0

	// SmallestMemory is the smallest possible MemoryLevel.  As MemoryLevel
	// increases from 1 (SmallestMemory) to 9 (FastestMemory), the amount
	// of memory used doubles with each increase.
	SmallestMemory MemoryLevel = 1

	// FastestMemory is the largest possible MemoryLevel.  As MemoryLevel
	// increases from 1 (SmallestMemory) to 9 (FastestMemory), the amount
	// of memory used doubles with each increase.
	FastestMemory MemoryLevel = 9
)

// IsValid returns true if mlevel is a valid MemoryLevel constant.
func (mlevel MemoryLevel) IsValid() bool {
	return mlevel >= DefaultMemory && mlevel <= FastestMemory
}

// GoString returns the Go string representation of this MemoryLevel constant.
func (mlevel MemoryLevel) GoString() string {
	if mlevel == DefaultMemory {
		return "DefaultMemory"
	}
	return fmt.Sprintf("MemoryLevel(%d)", uint(mlevel))
}

// String returns the string representation of this MemoryLevel constant.
func (mlevel MemoryLevel) String() string {
	if mlevel == DefaultMemory {
		return strDefault
	}
	return fmt.Sprintf("%d", uint(mlevel))
}

// MarshalJSON returns the JSON representation of this MemoryLevel constant.
func (mlevel MemoryLevel) MarshalJSON() ([]byte, error) {
	return json.Marshal(uint(mlevel))
}

// Parse parses a string representation of a MemoryLevel constant.
func (mlevel *MemoryLevel) Parse(str string) error {
	if strings.EqualFold(str, strDefault) {
		*mlevel = DefaultMemory
		return nil
	}

	u64, err := strconv.ParseUint(str, 10, 8)
	if err != nil {
		*mlevel = DefaultMemory
		return err
	}
	if u64 > 9 {
		*mlevel = DefaultMemory
		return fmt.Errorf("value %d is out of range", u64)
	}
	*mlevel = MemoryLevel(u64)
	return nil
}

var _ fmt.GoStringer = MemoryLevel(0)
var _ fmt.Stringer = MemoryLevel(0)
