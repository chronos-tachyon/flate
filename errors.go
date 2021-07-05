package flate

import (
	"fmt"
)

// CorruptInputError is returned when the stream being decompressed contains
// data that violates the compression format standard.
type CorruptInputError struct {
	OffsetTotal  uint64
	OffsetStream uint64
	Problem      string
}

// Error fulfills the error interface.
func (err CorruptInputError) Error() string {
	return fmt.Sprintf("corrupt input at/near byte offset %d: %s", err.OffsetStream, err.Problem)
}

var _ error = CorruptInputError{}
