package flate

import (
	"sync"

	"github.com/chronos-tachyon/assert"
)

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
