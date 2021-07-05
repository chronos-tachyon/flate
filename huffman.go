package flate

import (
	"encoding/json"
	"fmt"

	"github.com/chronos-tachyon/huffman"
)

const (
	logicalNumLLCodes  = 286
	logicalNumDCodes   = 30
	logicalNumXCodes   = 15 //nolint:deadcode,varcheck
	physicalNumLLCodes = 288
	physicalNumDCodes  = 32
	physicalNumXCodes  = 19
)

var scramble = [physicalNumXCodes]byte{16, 17, 18, 0, 8, 7, 9, 6, 10, 5, 11, 4, 12, 3, 13, 2, 14, 1, 15}

var (
	gFixedHuffmanDecoderLL huffman.Decoder
	gFixedHuffmanDecoderD  huffman.Decoder
	gFixedHuffmanEncoderLL huffman.Encoder
	gFixedHuffmanEncoderD  huffman.Encoder
)

func init() {
	// https://www.rfc-editor.org/rfc/rfc1951.html - Section 3.2.6
	sizes := make([]byte, physicalNumLLCodes)
	for i := 0; i < 144; i++ {
		sizes[i] = 8
	}
	for i := 144; i < 256; i++ {
		sizes[i] = 9
	}
	for i := 256; i < 280; i++ {
		sizes[i] = 7
	}
	for i := 280; i < 288; i++ {
		sizes[i] = 8
	}
	if err := gFixedHuffmanDecoderLL.Init(sizes); err != nil {
		panic(fmt.Errorf("failed to initialize gFixedHuffmanDecoderLL: %w", err))
	}
	if err := gFixedHuffmanEncoderLL.InitFromSizes(sizes); err != nil {
		panic(fmt.Errorf("failed to initialize gFixedHuffmanEncoderLL: %w", err))
	}

	sizes = sizes[:physicalNumDCodes]
	for i := 0; i < physicalNumDCodes; i++ {
		sizes[i] = 5
	}
	if err := gFixedHuffmanDecoderD.Init(sizes); err != nil {
		panic(fmt.Errorf("failed to initialize gFixedHuffmanDecoderD: %w", err))
	}
	if err := gFixedHuffmanEncoderD.InitFromSizes(sizes); err != nil {
		panic(fmt.Errorf("failed to initialize gFixedHuffmanEncoderD: %w", err))
	}
}

func getFixedHuffDecoders() (*huffman.Decoder, *huffman.Decoder) {
	return &gFixedHuffmanDecoderLL, &gFixedHuffmanDecoderD
}

func getFixedHuffEncoders() (*huffman.Encoder, *huffman.Encoder) {
	return &gFixedHuffmanEncoderLL, &gFixedHuffmanEncoderD
}

// SizeList represents a list of symbol sizes in a Canonical Huffman Code.
type SizeList []byte

// MarshalJSON returns the JSON representation of this SizeList, as a JSON
// Array of JSON Numbers.
func (sizelist SizeList) MarshalJSON() ([]byte, error) {
	var arr []uint
	if sizelist != nil {
		arr = make([]uint, len(sizelist))
		for index, size := range sizelist {
			arr[index] = uint(size)
		}
	}
	return json.Marshal(arr)
}
