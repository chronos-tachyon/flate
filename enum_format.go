package flate

import (
	"fmt"

	"github.com/chronos-tachyon/enumhelper"
)

// Format indicates the file format to be written (Writer) or expected to be
// read (Reader).
type Format byte

const (
	// DefaultFormat requests that Reader autodetect the file format, and
	// requests that Writer use the default format (currently ZlibFormat).
	DefaultFormat Format = iota

	// RawFormat indicates that a raw DEFLATE stream is in use, with no
	// headers or footers.
	RawFormat

	// ZlibFormat indicates that a zlib stream (RFC 1950) is in use.
	ZlibFormat

	// GZIPFormat indicates that a gzip stream (RFC 1952) is in use.
	GZIPFormat
)

var formatData = []enumhelper.EnumData{
	{GoName: "DefaultFormat", Name: "auto", Aliases: []string{strDefault}},
	{GoName: "RawFormat", Name: "raw"},
	{GoName: "ZlibFormat", Name: "zlib"},
	{GoName: "GZIPFormat", Name: "gzip"},
}

// IsValid returns true if f is a valid Format constant.
func (f Format) IsValid() bool {
	return f >= DefaultFormat && f <= GZIPFormat
}

// GoString returns the Go string representation of this Format constant.
func (f Format) GoString() string {
	return enumhelper.DereferenceEnumData("Format", formatData, uint(f)).GoName
}

// String returns the string representation of this Format constant.
func (f Format) String() string {
	return enumhelper.DereferenceEnumData("Format", formatData, uint(f)).Name
}

// MarshalJSON returns the JSON representation of this Format constant.
func (f Format) MarshalJSON() ([]byte, error) {
	return enumhelper.MarshalEnumToJSON("Format", formatData, uint(f))
}

// Parse parses a string representation of a Format constant.
func (f *Format) Parse(str string) error {
	value, err := enumhelper.ParseEnum("Format", formatData, str)
	*f = Format(value)
	return err
}

var _ fmt.GoStringer = Format(0)
var _ fmt.Stringer = Format(0)
