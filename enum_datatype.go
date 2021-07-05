package flate

import (
	"fmt"

	"github.com/chronos-tachyon/enumhelper"
)

// DataType indicates the autodetected content type of the compressed data,
// text (ASCII-compatible, including ISO-8859-1 or UTF-8) vs binary (anything
// else).
type DataType byte

const (
	// UnknownData indicates that autodetection was not performed.
	UnknownData DataType = iota

	// BinaryData indicates that autodetection found non-textual data.
	BinaryData

	// TextData indicates that autodetection found textual data.
	TextData
)

var dataTypeData = []enumhelper.EnumData{
	{GoName: "UnknownData", Name: "unknown"},
	{GoName: "BinaryData", Name: "binary"},
	{GoName: "TextData", Name: "text"},
}

// IsValid returns true if d is a valid DataType constant.
func (d DataType) IsValid() bool {
	return d >= UnknownData && d <= TextData
}

// GoString returns the Go string representation of this DataType constant.
func (d DataType) GoString() string {
	return enumhelper.DereferenceEnumData("DataType", dataTypeData, uint(d)).GoName
}

// String returns the string representation of this DataType constant.
func (d DataType) String() string {
	return enumhelper.DereferenceEnumData("DataType", dataTypeData, uint(d)).Name
}

// MarshalJSON returns the JSON representation of this DataType constant.
func (d DataType) MarshalJSON() ([]byte, error) {
	return enumhelper.MarshalEnumToJSON("DataType", dataTypeData, uint(d))
}

var _ fmt.GoStringer = DataType(0)
var _ fmt.Stringer = DataType(0)
