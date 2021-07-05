package flate

import (
	"fmt"

	"github.com/chronos-tachyon/enumhelper"
)

// EventType indicates the type of an Event.
type EventType byte

const (
	// StreamBeginEvent indicates that the beginning of a compressed stream
	// was detected.
	StreamBeginEvent EventType = iota

	// StreamHeaderEvent indicates that the stream header was successfully
	// processed.
	StreamHeaderEvent

	// BlockBeginEvent indicates that a block was detected.
	BlockBeginEvent

	// BlockTreesEvent indicates that the Huffman tree metadata for the
	// current block have been successfully processed.
	BlockTreesEvent

	// BlockEndEvent indicates that the data for the current block has been
	// successfully processed.
	BlockEndEvent

	// StreamEndEvent indicates that the end of the current stream was
	// detected.
	StreamEndEvent

	// StreamCloseEvent indicates that the stream footer was successfully
	// processed.
	StreamCloseEvent
)

var eventTypeData = []enumhelper.EnumData{
	{GoName: "StreamBeginEvent", Name: "stream-begin"},
	{GoName: "StreamHeaderEvent", Name: "stream-header"},
	{GoName: "BlockBeginEvent", Name: "block-begin"},
	{GoName: "BlockTreesEvent", Name: "block-trees"},
	{GoName: "BlockEndEvent", Name: "block-end"},
	{GoName: "StreamEndEvent", Name: "stream-end"},
	{GoName: "StreamCloseEvent", Name: "stream-close"},
}

// GoString returns the Go string representation of this EventType constant.
func (e EventType) GoString() string {
	return enumhelper.DereferenceEnumData("EventType", eventTypeData, uint(e)).GoName
}

// String returns the string representation of this EventType constant.
func (e EventType) String() string {
	return enumhelper.DereferenceEnumData("EventType", eventTypeData, uint(e)).Name
}

// MarshalJSON returns the JSON representation of this EventType constant.
func (e EventType) MarshalJSON() ([]byte, error) {
	return enumhelper.MarshalEnumToJSON("EventType", eventTypeData, uint(e))
}

var _ fmt.GoStringer = EventType(0)
var _ fmt.Stringer = EventType(0)
