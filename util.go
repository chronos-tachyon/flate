package flate

import (
	"hash"
	"io"
)

const strDefault = "default"

const bitsPerByte = 8

const bitsPerBlock = bytesPerBlock * bitsPerByte

func makeMask(shift byte) block {
	if shift == 0 {
		return 0
	} else if shift >= bitsPerBlock {
		return ^block(0)
	} else {
		return (block(1) << shift) - 1
	}
}

// type dummyHash32 {{{

type dummyHash32 struct{}

func (dummyHash32) Reset()                      {}
func (dummyHash32) BlockSize() int              { return 4 }
func (dummyHash32) Size() int                   { return 4 }
func (dummyHash32) Write(p []byte) (int, error) { return len(p), nil }
func (dummyHash32) Sum(p []byte) []byte         { return append(p, 0, 0, 0, 0) }
func (dummyHash32) Sum32() uint32               { return 0 }

var _ hash.Hash32 = dummyHash32{}

// }}}

// type eofReader {{{

type eofReader struct{}

func (eofReader) Read([]byte) (int, error) { return 0, io.EOF }

var _ io.Reader = eofReader{}

// }}}
