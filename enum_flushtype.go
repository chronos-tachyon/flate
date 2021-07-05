package flate

import (
	"fmt"

	"github.com/chronos-tachyon/enumhelper"
)

// FlushType indicates which type of Flush() operation is to be performed.
type FlushType byte

const (
	// BlockFlush completes the current block.  Some bits of the current
	// block may remain un-flushed in memory after this Flush().
	//
	// BlockFlush can impact your compression ratio by forcing the
	// premature end of the current block.
	//
	BlockFlush FlushType = iota

	// PartialFlush completes the current block and writes an empty
	// static-tree Huffman block (10 bits) as padding.  As the empty block
	// is more than 8 bits, all bits of the current block are guaranteed to
	// reach the underlying io.Writer by the completion of this Flush().
	//
	// Like BlockFlush, PartialFlush can impact your compression ratio.
	//
	PartialFlush

	// SyncFlush completes the current block and writes an empty
	// uncompressed block (35+ bits).  In addition to the guarantees made
	// by PartialFlush, this also guarantees that the next block will start
	// on a byte boundary.
	//
	// Like BlockFlush, SyncFlush can impact your compression ratio.
	//
	SyncFlush

	// FullFlush completes the current block, writes an empty uncompressed
	// block (35+ bits), and forgets the current sliding window.  In
	// addition to the guarantees made by PartialFlush and SyncFlush, this
	// also allows the stream reader to begin decompression at the start of
	// the next block without looking at any previous data.
	//
	// FullFlush will seriously degrade your compression ratio if not used
	// wisely, far more than BlockFlush would.
	//
	FullFlush

	// FinishFlush completes the current block, writes an empty
	// uncompressed block (35+ bits) that is marked BFINAL=1, writes the
	// footers for the current file format (if any), and forgets the
	// current sliding window.
	//
	// Any data written after this will begin a new concatenated stream.
	// Most users will want to call Close() instead.
	FinishFlush
)

var flushTypeData = []enumhelper.EnumData{
	{GoName: "BlockFlush", Name: "block"},
	{GoName: "PartialFlush", Name: "partial"},
	{GoName: "SyncFlush", Name: "sync"},
	{GoName: "FullFlush", Name: "full"},
	{GoName: "FinishFlush", Name: "finish"},
}

// IsValid returns true if f is a valid FlushType constant.
func (f FlushType) IsValid() bool {
	return f >= BlockFlush && f <= FinishFlush
}

// GoString returns the Go string representation of this FlushType constant.
func (f FlushType) GoString() string {
	return enumhelper.DereferenceEnumData("FlushType", flushTypeData, uint(f)).GoName
}

// String returns the string representation of this FlushType constant.
func (f FlushType) String() string {
	return enumhelper.DereferenceEnumData("FlushType", flushTypeData, uint(f)).Name
}

// MarshalJSON returns the JSON representation of this FlushType constant.
func (f FlushType) MarshalJSON() ([]byte, error) {
	return enumhelper.MarshalEnumToJSON("FlushType", flushTypeData, uint(f))
}

var _ fmt.GoStringer = FlushType(0)
var _ fmt.Stringer = FlushType(0)
