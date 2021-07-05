package flate

import (
	"fmt"

	"github.com/chronos-tachyon/enumhelper"
)

// Method indicates the compression method in use.
//
// Currently, only DEFLATE compression is supported.
//
type Method byte

const (
	// DeflateMethod indicates that DEFLATE compression should be used.
	DeflateMethod Method = iota

	// DefaultMethod requests that the default Method be used, which is
	// currently DeflateMethod.
	DefaultMethod = DeflateMethod
)

var methodData = []enumhelper.EnumData{
	{GoName: "DeflateMethod", Name: "deflate"},
}

// IsValid returns true if m is a valid Method constant.
func (m Method) IsValid() bool {
	return m == DeflateMethod
}

// GoString returns the Go string representation of this Method constant.
func (m Method) GoString() string {
	return enumhelper.DereferenceEnumData("Method", methodData, uint(m)).GoName
}

// String returns the string representation of this Method constant.
func (m Method) String() string {
	return enumhelper.DereferenceEnumData("Method", methodData, uint(m)).Name
}

// MarshalJSON returns the JSON representation of this Method constant.
func (m Method) MarshalJSON() ([]byte, error) {
	return enumhelper.MarshalEnumToJSON("Method", methodData, uint(m))
}

var _ fmt.GoStringer = Method(0)
var _ fmt.Stringer = Method(0)
