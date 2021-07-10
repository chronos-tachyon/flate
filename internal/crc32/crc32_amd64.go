// +build amd64

package crc32

import (
	"golang.org/x/sys/cpu"
)

//go:noescape
func ieeeCLMUL(sum uint32, p []byte) uint32

func archAvailable() bool {
	return cpu.X86.HasPCLMULQDQ && cpu.X86.HasSSE41
}

func archUpdate(sum uint32, p []byte) uint32 {
	length := uint(len(p))
	if length >= 64 {
		left := length & 0xf
		n := length - left
		sum = ^ieeeCLMUL(^sum, p[:n])
		p = p[n:]
		length -= n
	}
	return genericUpdate(sum, p)
}
