package flate

import (
	"github.com/chronos-tachyon/assert"
)

// Option represents a configuration option for Reader or Writer.
type Option func(*options)

type options struct {
	format   Format
	method   Method
	strategy Strategy
	clevel   CompressLevel
	mlevel   MemoryLevel
	wbits    WindowBits
	dict     []byte
	tracers  []Tracer
}

func (o *options) reset() {
	*o = options{
		format:   DefaultFormat,
		method:   DefaultMethod,
		strategy: DefaultStrategy,
		clevel:   DefaultCompression,
		mlevel:   DefaultMemory,
		wbits:    DefaultWindowBits,
		dict:     nil,
		tracers:  nil,
	}
}

func (o *options) apply(opts []Option) {
	for _, opt := range opts {
		opt(o)
	}
}

func (o *options) populateReaderDefaults() {
	if o.mlevel == DefaultMemory {
		o.mlevel = FastestMemory
	}
	if o.wbits == DefaultWindowBits {
		o.wbits = MaxWindowBits
	}
}

func (o *options) populateWriterDefaults() {
	o.populateReaderDefaults()
	if o.format == DefaultFormat {
		o.format = ZlibFormat
	}
	if o.clevel == DefaultCompression {
		o.clevel = 6
	}
}

// WithFormat specifies the Format to write (Writer) or expected to be read
// (Reader).
func WithFormat(format Format) Option {
	assert.Assertf(format.IsValid(), "invalid Format %d", uint(format))
	return func(o *options) { o.format = format }
}

// WithMethod specifies the Method to use (Writer).  Ignored by Reader.
func WithMethod(method Method) Option {
	assert.Assertf(method.IsValid(), "invalid Method %d", uint(method))
	return func(o *options) { o.method = method }
}

// WithStrategy specifies the Strategy to use (Writer).  Ignored by Reader.
func WithStrategy(strategy Strategy) Option {
	assert.Assertf(strategy.IsValid(), "invalid Strategy %d", uint(strategy))
	return func(o *options) { o.strategy = strategy }
}

// WithCompressLevel specifies the CompressLevel to use (Writer).  Ignored by Reader.
func WithCompressLevel(clevel CompressLevel) Option {
	assert.Assertf(clevel.IsValid(), "invalid CompressLevel %d", int(clevel))
	return func(o *options) { o.clevel = clevel }
}

// WithMemoryLevel specifies the MemoryLevel to use (Reader, Writer).
func WithMemoryLevel(mlevel MemoryLevel) Option {
	assert.Assertf(mlevel.IsValid(), "invalid MemoryLevel %d", uint(mlevel))
	return func(o *options) { o.mlevel = mlevel }
}

// WithWindowBits specifies the WindowBits to use (Writer) or the maximum
// WindowBits to expect (Reader).
func WithWindowBits(wbits WindowBits) Option {
	assert.Assertf(wbits.IsValid(), "invalid WindowBits %d", uint(wbits))
	return func(o *options) { o.wbits = wbits }
}

// WithDictionary specifies the pre-shared LZ77 dictionary to use (Writer) or
// to assume (Reader).  May specify nil to abandon a previously used pre-shared
// dictionary.
func WithDictionary(dict []byte) Option {
	assert.Assert(dict == nil || len(dict) > 0, "invalid zero-length dictionary; specify nil to omit the dictionary entirely")
	if dict != nil {
		tmp := make([]byte, len(dict))
		copy(tmp, dict)
		dict = tmp
	}
	return func(o *options) { o.dict = dict }
}

// WithTracers specifies the list of Tracer instances which will receive Events
// as compression (Writer) or decompression (Reader) proceeds.  Completely
// replaces any previous list.
func WithTracers(tracers ...Tracer) Option {
	for _, tr := range tracers {
		assert.NotNil(&tr)
	}
	if len(tracers) == 0 {
		tracers = nil
	} else {
		tmp := make([]Tracer, len(tracers))
		copy(tmp, tracers)
		tracers = tmp
	}
	return func(o *options) { o.tracers = tracers }
}
