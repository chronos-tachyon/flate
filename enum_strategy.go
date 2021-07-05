package flate

import (
	"fmt"

	"github.com/chronos-tachyon/enumhelper"
)

// Strategy indicates which compression strategy to use.  Each strategy
// represents a different compression algorithm, but all available algorithms
// are compatible with the standard decompression algorithm.
type Strategy byte

const (
	// DefaultStrategy indicates that the standard strategy be used.
	DefaultStrategy Strategy = iota

	// HuffmanOnlyStrategy indicates that LZ77 should not be used, only
	// Huffman coding.
	HuffmanOnlyStrategy

	// HuffmanRLEStrategy indicates that LZ77 should only be used for
	// run-length encoding (RLE), not for more complex history matching.
	HuffmanRLEStrategy

	// FixedStrategy indicates that all Huffman-encoded blocks should use
	// the static (pre-defined) Huffman encoding defined by the DEFLATE
	// standard, not the dynamic encodings that are optimal for each block.
	FixedStrategy
)

var strategyData = []enumhelper.EnumData{
	{GoName: "DefaultStrategy", Name: strDefault},
	{GoName: "HuffmanOnlyStrategy", Name: "huffman-only"},
	{GoName: "HuffmanRLEStrategy", Name: "huffman-rle"},
	{GoName: "FixedStrategy", Name: "fixed"},
}

// IsValid returns true if s is a valid Strategy constant.
func (s Strategy) IsValid() bool {
	return s >= DefaultStrategy && s <= FixedStrategy
}

// GoString returns the Go string representation of this Strategy constant.
func (s Strategy) GoString() string {
	return enumhelper.DereferenceEnumData("Strategy", strategyData, uint(s)).GoName
}

// String returns the string representation of this Strategy constant.
func (s Strategy) String() string {
	return enumhelper.DereferenceEnumData("Strategy", strategyData, uint(s)).Name
}

// MarshalJSON returns the JSON representation of this Strategy constant.
func (s Strategy) MarshalJSON() ([]byte, error) {
	return enumhelper.MarshalEnumToJSON("Strategy", strategyData, uint(s))
}

// Parse parses a string representation of a Strategy constant.
func (s *Strategy) Parse(str string) error {
	value, err := enumhelper.ParseEnum("Strategy", strategyData, str)
	*s = Strategy(value)
	return err
}

var _ fmt.GoStringer = Strategy(0)
var _ fmt.Stringer = Strategy(0)
