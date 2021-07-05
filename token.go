package flate

import (
	"fmt"
	"math"
	"math/bits"

	"github.com/chronos-tachyon/assert"
	"github.com/chronos-tachyon/huffman"
)

type tokenType byte

const (
	invalidToken tokenType = iota
	copyToken
	literalToken
	stopToken
	treeLenToken
	treeDupToken
	treeSZRToken
	treeLZRToken
)

type token struct {
	literalOrLength uint16
	distance        uint16
}

func makeCopyToken(length uint16, distance uint16) token {
	assert.Assertf(length >= 3, "copy length %d < minimum 3", length)
	assert.Assertf(length <= 258, "copy length %d > maximum 258", length)
	assert.Assertf(distance >= 1, "copy distance %d < minimum 1", distance)
	assert.Assertf(distance <= 32768, "copy distance %d > maximum 32768", distance)
	return token{literalOrLength: length, distance: distance}
}

func makeLiteralToken(ch byte) token {
	return token{literalOrLength: uint16(ch), distance: 0}
}

func makeStopToken() token {
	return token{literalOrLength: 256, distance: 0}
}

func makeTreeLenToken(size byte) token {
	assert.Assertf(size < 16, "symbol bit length %d >= 16", size)
	return token{literalOrLength: 512 + uint16(size), distance: 0}
}

func makeTreeDupToken(count uint) token {
	assert.Assertf(count >= 3, "symbol bit length copy count %d < minimum 3", count)
	assert.Assertf(count <= 6, "symbol bit length copy count %d > maximum 6", count)
	return token{literalOrLength: 1024 + uint16(count-3), distance: 0}
}

func makeTreeZeroRunToken(count uint) token {
	assert.Assertf(count >= 3, "symbol bit length zero count %d < minimum 3", count)
	assert.Assertf(count <= 138, "symbol bit length zero count %d > maximum 138", count)
	if count < 11 {
		return token{literalOrLength: 2048 + uint16(count-3), distance: 0}
	}
	return token{literalOrLength: 4096 + uint16(count-11), distance: 0}
}

func makeInvalidToken() token {
	return token{literalOrLength: math.MaxUint16, distance: 0}
}

func (t token) tokenType() tokenType {
	switch {
	case t.distance != 0:
		return copyToken
	case t.literalOrLength < 256:
		return literalToken
	case t.literalOrLength == 256:
		return stopToken
	case t.literalOrLength >= 512 && t.literalOrLength < (512+16):
		return treeLenToken
	case t.literalOrLength >= 1024 && t.literalOrLength < (1024+4):
		return treeDupToken
	case t.literalOrLength >= 2048 && t.literalOrLength < (2048+8):
		return treeSZRToken
	case t.literalOrLength >= 4096 && t.literalOrLength < (4096+128):
		return treeLZRToken
	default:
		assert.Raisef("invalid token literalOrLength=%d distance=%d", t.literalOrLength, t.distance)
		return invalidToken
	}
}

func (t token) symbolLL() (symbol huffman.Symbol, bitLen byte, bitBlock block) {
	switch {
	case t.distance == 0 && t.literalOrLength > 256:
		return huffman.InvalidSymbol, 0, 0

	case t.distance == 0 && t.literalOrLength == 256:
		return huffman.Symbol(256), 0, 0

	case t.distance == 0:
		return huffman.Symbol(t.literalOrLength), 0, 0

	case t.literalOrLength <= 2:
		assert.Raisef("copy length %d < minimum 3", t.literalOrLength)
		return huffman.InvalidSymbol, 0, 0

	case t.literalOrLength <= 10:
		return huffman.Symbol(254 + t.literalOrLength), 0, 0

	case t.literalOrLength <= 18:
		x := (t.literalOrLength - 11)
		y, z := (x / 2), (x % 2)
		return huffman.Symbol(265 + y), 1, block(z)

	case t.literalOrLength <= 34:
		x := (t.literalOrLength - 19)
		y, z := (x / 4), (x % 4)
		return huffman.Symbol(269 + y), 2, block(z)

	case t.literalOrLength <= 66:
		x := (t.literalOrLength - 35)
		y, z := (x / 8), (x % 8)
		return huffman.Symbol(273 + y), 3, block(z)

	case t.literalOrLength <= 130:
		x := (t.literalOrLength - 67)
		y, z := (x / 16), (x % 16)
		return huffman.Symbol(277 + y), 4, block(z)

	case t.literalOrLength <= 257:
		x := (t.literalOrLength - 131)
		y, z := (x / 32), (x % 32)
		return huffman.Symbol(281 + y), 5, block(z)

	case t.literalOrLength == 258:
		return huffman.Symbol(285), 0, 0

	default:
		assert.Raisef("copy length %d > maximum 258", t.literalOrLength)
		return huffman.InvalidSymbol, 0, 0
	}
}

func (t token) symbolD() (symbol huffman.Symbol, bitLen byte, bitBlock block) {
	switch {
	case t.distance == 0:
		return huffman.InvalidSymbol, 0, 0

	case t.distance <= 4:
		d := (t.distance - 1)
		return huffman.Symbol(0 + d), 0, 0

	case t.distance <= 32768:
		d := (t.distance - 1)
		k := 16 - bits.LeadingZeros16(d)
		code := k*2 - 1
		if bit := uint16(1) << (k - 2); (d & bit) == 0 {
			code--
		}
		size := byte((code / 2) - 1)
		mask := (uint16(1) << size) - 1
		bits := (d & mask)
		return huffman.Symbol(code), size, block(bits)

	default:
		assert.Raisef("copy distance %d > maximum 32768", t.distance)
		return huffman.InvalidSymbol, 0, 0
	}
}

func (t token) symbolX() (symbol huffman.Symbol, bitLen byte, bitBlock block) {
	switch {
	case t.distance != 0:
		return huffman.InvalidSymbol, 0, 0

	case t.literalOrLength < 512:
		return huffman.InvalidSymbol, 0, 0

	case t.literalOrLength >= 512 && t.literalOrLength < (512+16):
		return huffman.Symbol(t.literalOrLength - 512), 0, 0

	case t.literalOrLength >= 1024 && t.literalOrLength < (1024+4):
		return huffman.Symbol(16), 2, block(t.literalOrLength - 1024)

	case t.literalOrLength >= 2048 && t.literalOrLength < (2048+8):
		return huffman.Symbol(17), 3, block(t.literalOrLength - 2048)

	case t.literalOrLength >= 4096 && t.literalOrLength < (4096+128):
		return huffman.Symbol(18), 7, block(t.literalOrLength - 4096)

	default:
		return huffman.InvalidSymbol, 0, 0
	}
}

func (t token) encodeLLD(bw bitwriter, hLL *huffman.Encoder, hD *huffman.Encoder) bool {
	symLL, sizeLL, bitsLL := t.symbolLL()
	symD, sizeD, bitsD := t.symbolD()

	if symLL >= 0 {
		hc := hLL.Encode(symLL)
		if !bw.outputBitsWriteHC(hc) {
			return false
		}

		if sizeLL != 0 {
			if !bw.outputBitsWrite(sizeLL, bitsLL) {
				return false
			}
		}

		if symD >= 0 {
			hc := hD.Encode(symD)
			if !bw.outputBitsWriteHC(hc) {
				return false
			}

			if sizeD != 0 {
				if !bw.outputBitsWrite(sizeD, bitsD) {
					return false
				}
			}
		}
	}

	return true
}

func (t token) encodeX(bw bitwriter, hX *huffman.Encoder) bool {
	symX, sizeX, bitsX := t.symbolX()

	if symX >= 0 {
		hc := hX.Encode(symX)
		if !bw.outputBitsWriteHC(hc) {
			return false
		}

		if sizeX != 0 {
			if !bw.outputBitsWrite(sizeX, bitsX) {
				return false
			}
		}
	}

	return true
}

func (t token) String() string {
	switch t.tokenType() {
	case copyToken:
		return fmt.Sprintf("[copy token: distance=%d length=%d]", t.distance, t.literalOrLength)

	case literalToken:
		return fmt.Sprintf("[literal token: %#02x]", t.literalOrLength)

	case stopToken:
		return "[stop token]"

	case treeLenToken:
		return fmt.Sprintf("[tree len token: length=%d]", t.literalOrLength-512)

	case treeDupToken:
		return fmt.Sprintf("[tree dup token: count=%d]", t.literalOrLength-1024+3)

	case treeSZRToken:
		return fmt.Sprintf("[tree short zero repeat token: count=%d]", t.literalOrLength-2048+3)

	case treeLZRToken:
		return fmt.Sprintf("[tree long zero repeat token: count=%d]", t.literalOrLength-4096+11)

	default:
		return fmt.Sprintf("[invalid token: ll=%d d=%d]", t.literalOrLength, t.distance)
	}
}
