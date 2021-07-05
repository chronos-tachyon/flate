package flate

import (
	"github.com/chronos-tachyon/assert"
	"github.com/rs/zerolog"
)

// Tracer is an interface which callers can implement in order to receive
// Events.  Events provide feedback on the progress of the compression or
// decompression operation.
type Tracer interface {
	OnEvent(Event)
}

// Event is a collection of fields that provide feedback on the progress of the
// compression or decompression operation in progress.  Events are provided to
// Tracers registered with a Reader or Writer.
type Event struct {
	Type              EventType
	InputBytesTotal   uint64
	InputBytesStream  uint64
	OutputBytesTotal  uint64
	OutputBytesStream uint64
	NumStreams        uint
	Format            Format
	Header            *Header
	Block             *BlockEvent
	Trees             *TreesEvent
	Footer            *FooterEvent
}

// BlockEvent is a sub-struct that is only present for BlockFooEvent.
type BlockEvent struct {
	Type    BlockType
	IsFinal bool
}

// TreesEvent is a sub-struct that is only present for BlockTreesEvent.
type TreesEvent struct {
	CodeCount          uint16
	LiteralLengthCount uint16
	DistanceCount      uint16

	CodeSizes          SizeList
	LiteralLengthSizes SizeList
	DistanceSizes      SizeList
}

// FooterEvent is a sub-struct that is only present for StreamEndEvent.
type FooterEvent struct {
	Adler32 Checksum32
	CRC32   Checksum32
}

// type NoOpTracer {{{

// NoOpTracer is an implementation of Tracer that does nothing.
type NoOpTracer struct{}

// OnEvent fulfills Tracer.
func (NoOpTracer) OnEvent(event Event) {}

var _ Tracer = NoOpTracer{}

// }}}

// type TracerFunc {{{

// TracerFunc is an implementation of Tracer that calls a function.
type TracerFunc func(Event)

// OnEvent fulfills Tracer.
func (tr TracerFunc) OnEvent(event Event) {
	tr(event)
}

var _ Tracer = TracerFunc(nil)

// }}}

// type captureHeaderTracer {{{

// CaptureHeader returns a Tracer implementation which will fill the pointed-to
// Header object when StreamHeaderEvent is encountered.
func CaptureHeader(ptr *Header) Tracer {
	assert.NotNil(&ptr)
	return captureHeaderTracer{ptr: ptr}
}

type captureHeaderTracer struct {
	ptr *Header
}

// OnEvent fulfills Tracer.
func (tr captureHeaderTracer) OnEvent(event Event) {
	if event.Type == StreamHeaderEvent && event.Header != nil {
		*tr.ptr = *event.Header
	}
}

var _ Tracer = captureHeaderTracer{}

// }}}

// type logTracer {{{

// Log returns a Tracer implementation which will log each Event at Trace
// priority.
func Log(logger zerolog.Logger) Tracer {
	return logTracer{logger: logger}
}

type logTracer struct {
	logger zerolog.Logger
}

// OnEvent fulfills Tracer.
func (tr logTracer) OnEvent(event Event) {
	tr.logger.Trace().
		Interface("event", event).
		Msg("OnEvent")
}

var _ Tracer = logTracer{}

// }}}
