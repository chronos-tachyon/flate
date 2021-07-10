package adler32

import (
	"encoding/binary"
	"hash"
)

const Size = 4

const modulus = 65521

const nmax = 5552

func Update(sum uint32, p []byte) uint32 {
	s1, s2 := (sum & 0xffff), (sum >> 16)
	length := uint(len(p))
	for length > 0 {
		var q []byte
		if length > nmax {
			p, q = p[:nmax], p[nmax:]
			length = nmax
		}
		for length >= 4 {
			s1 += uint32(p[0])
			s2 += s1
			s1 += uint32(p[1])
			s2 += s1
			s1 += uint32(p[2])
			s2 += s1
			s1 += uint32(p[3])
			s2 += s1
			p = p[4:]
			length -= 4
		}
		for _, ch := range p {
			s1 += uint32(ch)
			s2 += s1
		}
		s1 %= modulus
		s2 %= modulus
		p = q
		length = uint(len(q))
	}
	return (s2 << 16) | s1
}

func Checksum(p []byte) uint32 {
	var h Hash
	h.Reset()
	h.Write(p)
	return h.Sum32()
}

type Hash struct {
	sum uint32
}

func New() *Hash {
	return &Hash{sum: 1}
}

func (h *Hash) Size() int      { return Size }
func (h *Hash) BlockSize() int { return 1 }

func (h *Hash) Reset() {
	h.sum = 1
}

func (h *Hash) Write(p []byte) (int, error) {
	h.sum = Update(h.sum, p)
	return len(p), nil
}

func (h *Hash) Sum(slice []byte) []byte {
	var tmp [Size]byte
	binary.BigEndian.PutUint32(tmp[:], h.Sum32())
	return append(slice, tmp[:]...)
}

func (h *Hash) Sum32() uint32 {
	return h.sum
}

var _ hash.Hash32 = (*Hash)(nil)
