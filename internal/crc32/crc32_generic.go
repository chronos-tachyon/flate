// +build !amd64,!arm64

package crc32

func archAvailable() bool {
	return false
}

func archUpdate(sum uint32, p []byte) uint32 {
	return genericUpdate(sum, p)
}
