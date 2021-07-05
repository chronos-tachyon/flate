package flate

import (
	"bytes"
	"strings"
	"sync"

	"github.com/chronos-tachyon/assert"
)

var sbPool = sync.Pool{
	New: func() interface{} {
		sb := new(strings.Builder)
		sb.Grow(256)
		return sb
	},
}

func takeStringsBuilder() *strings.Builder {
	return sbPool.Get().(*strings.Builder)
}

func giveStringsBuilder(sb *strings.Builder) {
	assert.NotNil(&sb)
	sb.Reset()
	sbPool.Put(sb)
}

var bbPool = sync.Pool{
	New: func() interface{} {
		bb := new(bytes.Buffer)
		bb.Grow(256)
		return bb
	},
}

func takeBytesBuffer() *bytes.Buffer {
	return bbPool.Get().(*bytes.Buffer)
}

func giveBytesBuffer(bb *bytes.Buffer) {
	assert.NotNil(&bb)
	bb.Reset()
	bbPool.Put(bb)
}

var tokenPool = sync.Pool{
	New: func() interface{} {
		ptr := new([]token)
		*ptr = make([]token, 0, 256)
		return ptr
	},
}

func takeTokens() *[]token {
	return tokenPool.Get().(*[]token)
}

func giveTokens(ptr *[]token) {
	assert.NotNil(&ptr)
	assert.NotNil(ptr)
	*ptr = (*ptr)[:0]
	tokenPool.Put(ptr)
}
