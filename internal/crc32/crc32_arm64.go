// +build arm64

package crc32

import (
	"golang.org/x/sys/cpu"
)

//go:noescape
func ieeeUpdate(sum uint32, p []byte) uint32

func archAvailable() bool {
	return cpu.ARM64.HasCRC32
}

func archUpdate(sum uint32, p []byte) uint32 {
	return ^ieeeUpdate(^sum, p)
}
