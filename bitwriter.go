package flate

import (
	"encoding/binary"

	"github.com/chronos-tachyon/assert"
	"github.com/chronos-tachyon/huffman"
)

type bitwriter interface {
	outputBufferWrite([]byte) bool
	outputBufferWriteU16(binary.ByteOrder, uint16) bool
	outputBufferWriteU32(binary.ByteOrder, uint32) bool
	outputBitsWrite(byte, block) bool
	outputBitsWriteHC(huffman.Code) bool
	outputBitsFlush() bool
	sendEvent(Event)
}

type bitcounter struct {
	numBits uint64
}

func (bc bitcounter) length() uint64 {
	return bc.numBits
}

func (bc *bitcounter) outputBufferWrite(p []byte) bool {
	bc.outputBitsFlush()
	bc.numBits += uint64(len(p)) << 3
	return true
}

func (bc *bitcounter) outputBufferWriteU16(bo binary.ByteOrder, x uint16) bool {
	bc.outputBitsFlush()
	bc.numBits += 16
	return true
}

func (bc *bitcounter) outputBufferWriteU32(bo binary.ByteOrder, x uint32) bool {
	bc.outputBitsFlush()
	bc.numBits += 32
	return true
}

func (bc *bitcounter) outputBitsWrite(size byte, bits block) bool {
	assert.Assertf(size <= bitsPerBlock, "size %d > bitsPerBlock %d", size, bitsPerBlock)
	bc.numBits += uint64(size)
	return true
}

func (bc *bitcounter) outputBitsWriteHC(hc huffman.Code) bool {
	return bc.outputBitsWrite(hc.Size, block(hc.Bits))
}

func (bc *bitcounter) outputBitsFlush() bool {
	remainder := (bc.numBits & 7)
	if remainder != 0 {
		bc.numBits += (8 - remainder)
	}
	return true
}

func (bc *bitcounter) sendEvent(Event) {
}

var _ bitwriter = (*bitcounter)(nil)
