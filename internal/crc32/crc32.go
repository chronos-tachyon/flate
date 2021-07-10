package crc32

import (
	"encoding/binary"
	"hash"
	"sync"
)

const Size = 4

const ieeePolynomial = 0xedb88320

type basicTable [256]uint32

type slicingTable [8]basicTable

var ieee slicingTable

var gArchOnce sync.Once
var gArchOkay bool

func init() {
	for i := uint32(0); i < 256; i++ {
		sum := i
		for j := 0; j < 8; j++ {
			if (sum & 1) == 1 {
				sum = (sum >> 1) ^ ieeePolynomial
			} else {
				sum >>= 1
			}
		}
		ieee[0][i] = sum
	}
	for i := uint32(0); i < 256; i++ {
		sum := ieee[0][i]
		for j := 1; j < 8; j++ {
			sum = ieee[0][sum&0xff] ^ (sum >> 8)
			ieee[j][i] = sum
		}
	}
}

func Update(sum uint32, p []byte) uint32 {
	gArchOnce.Do(func() {
		gArchOkay = archAvailable()
	})
	if gArchOkay {
		return archUpdate(sum, p)
	}
	return genericUpdate(sum, p)
}

func genericUpdate(sum uint32, p []byte) uint32 {
	length := uint(len(p))
	if length == 0 {
		return sum
	}
	sum = ^sum
	if length >= 16 {
		for length >= 8 {
			sum ^= binary.LittleEndian.Uint32(p)
			sum = (ieee[0][p[7]] ^
				ieee[1][p[6]] ^
				ieee[2][p[5]] ^
				ieee[3][p[4]] ^
				ieee[4][sum>>24] ^
				ieee[5][(sum>>16)&0xff] ^
				ieee[6][(sum>>8)&0xff] ^
				ieee[7][sum&0xff])
			p = p[8:]
			length -= 8
		}
	}
	for _, ch := range p {
		sum = ieee[0][byte(sum)^ch] ^ (sum >> 8)
	}
	sum = ^sum
	return sum
}

type Hash struct {
	sum uint32
}

func New() *Hash {
	return new(Hash)
}

func (h *Hash) Size() int      { return Size }
func (h *Hash) BlockSize() int { return 1 }

func (h *Hash) Reset() {
	h.sum = 0
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
