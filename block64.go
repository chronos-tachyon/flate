// +build !386,!arm

package flate

import (
	"encoding/binary"
	"math/bits"
)

const bytesPerBlock = 8

type block uint64

func bytesFromBlock(byteOrder binary.ByteOrder, p []byte, x block) {
	byteOrder.PutUint64(p, uint64(x))
}

//nolint:deadcode,unused
func bytesAsBlock(byteOrder binary.ByteOrder, p []byte) block {
	return block(byteOrder.Uint64(p))
}

//nolint:deadcode,unused
func reverseBlock(x block) block {
	return block(bits.Reverse64(uint64(x)))
}
