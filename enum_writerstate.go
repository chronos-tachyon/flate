package flate

import (
	"fmt"

	"github.com/chronos-tachyon/enumhelper"
)

type writerState byte

const (
	// noStreamWriterState: we (1) are NOT inside a stream, and (2) have
	// written 0 streams total.
	noStreamWriterState writerState = iota

	// openStreamWriterState: we (1) are inside a stream, (2) have written
	// 0 or more non-final blocks, and (3) have not written any final
	// blocks.
	openStreamWriterState

	// closingStreamWriterState: we (1) are inside a stream, (2) have
	// written 0 or more non-final blocks, and (3) have written exactly 1
	// final block.
	closingStreamWriterState

	// closedStreamWriterState: we (1) are NOT inside a stream, (2) have
	// written 1 or more streams total, and (3) are ready to Close or to
	// open another stream.
	closedStreamWriterState

	// errorWriterState: our written state should be assumed corrupt, and
	// the only valid action is to Close.
	errorWriterState

	// closedWriterState: Close has been called, and the only valid action
	// is to Reset.
	closedWriterState
)

var writerStateData = []enumhelper.EnumData{
	{GoName: "noStreamWriterState", Name: "noStream"},
	{GoName: "openStreamWriterState", Name: "openStream"},
	{GoName: "closingStreamWriterState", Name: "closingStream"},
	{GoName: "closedStreamWriterState", Name: "closedStream"},
	{GoName: "errorWriterState", Name: "error"},
	{GoName: "closedWriterState", Name: "closed"},
}

func (s writerState) GoString() string {
	return enumhelper.DereferenceEnumData("writerState", writerStateData, uint(s)).GoName
}

func (s writerState) String() string {
	return enumhelper.DereferenceEnumData("writerState", writerStateData, uint(s)).Name
}

func (s writerState) MarshalJSON() ([]byte, error) {
	return enumhelper.MarshalEnumToJSON("writerState", writerStateData, uint(s))
}

var _ fmt.GoStringer = writerState(0)
var _ fmt.Stringer = writerState(0)
