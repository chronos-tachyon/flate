package flate

import (
	"fmt"

	"github.com/chronos-tachyon/enumhelper"
)

// BlockType indicates the type of a DEFLATE-compressed block.
type BlockType byte

const (
	// InvalidBlock is a dummy value indicating an invalid block.
	InvalidBlock BlockType = iota

	// StoredBlock indicates a stored (uncompressed) block.
	StoredBlock

	// StaticBlock indicates a Huffman-compressed block that uses the
	// static (pre-defined) Huffman encoding.
	StaticBlock

	// DynamicBlock indicates a Huffman-compressed block that uses a
	// dynamic Huffman encoding produced just for this block.
	DynamicBlock
)

var blockTypeData = []enumhelper.EnumData{
	{GoName: "InvalidBlock", Name: "invalid"},
	{GoName: "StoredBlock", Name: "stored"},
	{GoName: "StaticBlock", Name: "static"},
	{GoName: "DynamicBlock", Name: "dynamic"},
}

// GoString returns the Go string representation of this BlockType constant.
func (b BlockType) GoString() string {
	return enumhelper.DereferenceEnumData("BlockType", blockTypeData, uint(b)).GoName
}

// String returns the string representation of this BlockType constant.
func (b BlockType) String() string {
	return enumhelper.DereferenceEnumData("BlockType", blockTypeData, uint(b)).Name
}

// MarshalJSON returns the JSON representation of this BlockType constant.
func (b BlockType) MarshalJSON() ([]byte, error) {
	return enumhelper.MarshalEnumToJSON("BlockType", blockTypeData, uint(b))
}

var _ fmt.GoStringer = BlockType(0)
var _ fmt.Stringer = BlockType(0)
