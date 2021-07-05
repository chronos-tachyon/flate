// +build 386 arm

package flate

import (
	"encoding/binary"
	"math/bits"
)

const bytesPerBlock = 4

type block uint32

func bytesFromBlock(byteOrder binary.ByteOrder, p []byte, x block) {
	byteOrder.PutUint32(p, uint32(x))
}

//nolint:deadcode,unused
func bytesAsBlock(byteOrder binary.ByteOrder, p []byte) block {
	return block(byteOrder.Uint32(p))
}

//nolint:deadcode,unused
func reverseBlock(x block) block {
	return block(bits.Reverse32(uint32(x)))
}
