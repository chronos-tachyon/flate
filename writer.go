package flate

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"strings"
	"sync"
	"syscall"

	"github.com/chronos-tachyon/assert"
	buffer "github.com/chronos-tachyon/buffer/v3"
	"github.com/chronos-tachyon/huffman"
	"github.com/hashicorp/go-multierror"

	"github.com/chronos-tachyon/flate/internal/adler32"
	"github.com/chronos-tachyon/flate/internal/crc32"
)

type flushWriter interface {
	io.Writer
	Flush() error
}

type syncWriter interface {
	io.Writer
	Sync() error
}

// Writer wraps an io.Writer and compresses the data which flows through it.
type Writer struct {
	mu sync.Mutex

	format   Format
	method   Method
	strategy Strategy
	clevel   CompressLevel
	mlevel   MemoryLevel
	wbits    WindowBits
	dict     []byte
	tracers  []Tracer

	cfg compressConfig

	w                 io.Writer
	err               error
	inputBytesAdler32 adler32.Hash
	inputBytesCRC32   crc32.Hash
	outputBytesCRC32  crc32.Hash
	hybrid            buffer.LZ77
	output            buffer.Buffer
	inputBytesTotal   uint64
	inputBytesStream  uint64
	outputBytesTotal  uint64
	outputBytesStream uint64
	numStreams        uint
	obBlock           block
	obLen             byte
	state             writerState

	header Header
}

// NewWriter constructs and returns a new Writer with the given io.Writer and
// options.
func NewWriter(w io.Writer, opts ...Option) *Writer {
	var o options
	o.reset()
	o.apply(opts)
	o.populateWriterDefaults()

	fw := &Writer{
		format:   o.format,
		method:   o.method,
		strategy: o.strategy,
		clevel:   o.clevel,
		mlevel:   o.mlevel,
		wbits:    o.wbits,
		dict:     o.dict,
		tracers:  o.tracers,

		w: w,
	}

	fw.hybrid.Init(fw.hybridOptions())
	fw.output.Init(fw.outputNumBits())
	fw.cfg = fw.compressConfig()

	return fw
}

func (fw *Writer) outputNumBits() uint {
	return uint(fw.mlevel + 7) // 8 .. 16 ⇒ [256 bytes .. 64 kibibytes]
}

func (fw *Writer) hybridOptions() buffer.LZ77Options {
	defaultOpts := buffer.LZ77Options{
		BufferNumBits:     uint(fw.mlevel + 8),  //  9 .. 17 ⇒ [ 512 bytes   .. 128 kibibytes]
		WindowNumBits:     uint(fw.wbits),       //  8 .. 15 ⇒ [ 256 bytes   ..  32 kibibytes]
		HashNumBits:       uint(fw.mlevel + 11), // 12 .. 20 ⇒ [4096 entries ..   1 mebientries]
		MinMatchLength:    4,
		MaxMatchLength:    258,
		HasMinMatchLength: true,
		HasMaxMatchLength: true,
	}
	switch {
	case fw.strategy == HuffmanOnlyStrategy:
		defaultOpts.WindowNumBits = 8
		defaultOpts.MinMatchLength = 0
		defaultOpts.MaxMatchLength = 0
		defaultOpts.MaxMatchDistance = 0
		defaultOpts.HasMaxMatchDistance = true
	case fw.strategy == HuffmanRLEStrategy:
		defaultOpts.WindowNumBits = 8
		defaultOpts.MinMatchLength = 3
		defaultOpts.MaxMatchDistance = 1
		defaultOpts.HasMaxMatchDistance = true
	}
	return defaultOpts
}

func (fw *Writer) compressConfig() compressConfig {
	switch {
	case fw.clevel == 0:
		return compressConfigs[2]
	case fw.strategy == HuffmanOnlyStrategy:
		return compressConfigs[1]
	case fw.strategy == HuffmanRLEStrategy:
		return compressConfigs[0]
	case fw.clevel > 0:
		return compressConfigs[2+fw.clevel]
	default:
		return compressConfigs[2+6]
	}
}

// Format returns the Format which this Writer uses.
func (fw *Writer) Format() Format {
	fw.mu.Lock()
	format := fw.format
	fw.mu.Unlock()
	return format
}

// Method returns the Method which this Writer uses.
func (fw *Writer) Method() Method {
	fw.mu.Lock()
	method := fw.method
	fw.mu.Unlock()
	return method
}

// Strategy returns the Strategy which this Writer uses.
func (fw *Writer) Strategy() Strategy {
	fw.mu.Lock()
	strategy := fw.strategy
	fw.mu.Unlock()
	return strategy
}

// CompressLevel returns the CompressLevel which this Writer uses.
func (fw *Writer) CompressLevel() CompressLevel {
	fw.mu.Lock()
	clevel := fw.clevel
	fw.mu.Unlock()
	return clevel
}

// MemoryLevel returns the MemoryLevel which this Writer uses.
func (fw *Writer) MemoryLevel() MemoryLevel {
	fw.mu.Lock()
	mlevel := fw.mlevel
	fw.mu.Unlock()
	return mlevel
}

// WindowBits returns the WindowBits which this Writer uses.
func (fw *Writer) WindowBits() WindowBits {
	fw.mu.Lock()
	wbits := fw.wbits
	fw.mu.Unlock()
	return wbits
}

// Dict returns the pre-shared LZ77 dictionary which this Writer uses, or nil
// if no such dictionary is in use.
func (fw *Writer) Dict() []byte {
	var dict []byte
	fw.mu.Lock()
	if len(fw.dict) != 0 {
		dict = make([]byte, len(fw.dict))
		copy(dict, fw.dict)
	}
	fw.mu.Unlock()
	return dict
}

// Tracers returns the Tracers which this Writer uses.
func (fw *Writer) Tracers() []Tracer {
	var tracers []Tracer
	fw.mu.Lock()
	if len(fw.tracers) != 0 {
		tracers = make([]Tracer, len(fw.tracers))
		copy(tracers, fw.tracers)
	}
	fw.mu.Unlock()
	return tracers
}

// UnderlyingWriter returns the io.Writer which this Writer uses.
func (fw *Writer) UnderlyingWriter() io.Writer {
	fw.mu.Lock()
	w := fw.w
	fw.mu.Unlock()
	return w
}

// Reset re-initializes this Writer with the given io.Writer and options.  Any
// options given here are merged with all previous options.
func (fw *Writer) Reset(w io.Writer, opts ...Option) {
	assert.NotNil(&w)
	for _, opt := range opts {
		assert.NotNil(&opt)
	}

	fw.mu.Lock()
	defer fw.mu.Unlock()

	if fw.state == openStreamWriterState || fw.state == closingStreamWriterState {
		assert.Raisef("invalid state %#v -- cannot Reset in the middle of a stream", fw.state)
	}

	fw.w = w
	fw.err = nil
	fw.state = noStreamWriterState

	fw.hybrid.Clear()
	fw.output.Clear()

	if len(opts) == 0 {
		return
	}

	var o options
	o.reset()
	o.format = fw.format
	o.method = fw.method
	o.strategy = fw.strategy
	o.clevel = fw.clevel
	o.mlevel = fw.mlevel
	o.wbits = fw.wbits
	o.dict = fw.dict
	o.tracers = fw.tracers
	o.apply(opts)
	o.populateWriterDefaults()

	fw.format = o.format
	fw.method = o.method
	fw.strategy = o.strategy
	fw.clevel = o.clevel
	fw.mlevel = o.mlevel
	fw.wbits = o.wbits
	fw.dict = o.dict
	fw.tracers = o.tracers

	oldHO := fw.hybrid.Options()
	newHO := fw.hybridOptions()
	if !oldHO.Equal(newHO) {
		fw.hybrid.Init(newHO)
	}

	oldONB := fw.output.NumBits()
	newONB := fw.outputNumBits()
	if oldONB != newONB {
		fw.output.Init(newONB)
	}

	fw.cfg = fw.compressConfig()
}

// SetHeader sets a custom gzip header when writing in GZIPFormat.
func (fw *Writer) SetHeader(header Header) error {
	var errlist []error

	errlist = checkHeaderFileName(header, errlist)
	errlist = checkHeaderComment(header, errlist)
	errlist = checkHeaderLastModified(header, errlist)
	errlist = checkHeaderOSType(header, errlist)
	errlist = checkHeaderExtraData(header, errlist)

	if len(errlist) == 0 {
		fw.mu.Lock()
		fw.header = header
		fw.mu.Unlock()
		return nil
	}

	if len(errlist) == 1 {
		return errlist[0]
	}

	return &multierror.Error{Errors: errlist}
}

func checkHeaderFileName(header Header, errlist []error) []error {
	if len(header.FileName) >= 256 {
		errlist = append(errlist, errors.New("Header.FileName is longer than 256 bytes"))
	}
	if index := strings.IndexByte(header.FileName, 0x00); index >= 0 {
		errlist = append(errlist, errors.New("Header.FileName contains embedded NUL byte"))
	}
	if index := strings.IndexByte(header.FileName, 0x2f); index >= 0 {
		errlist = append(errlist, errors.New("Header.FileName contains embedded '/' byte"))
	}
	if index := strings.IndexByte(header.FileName, 0x5c); index >= 0 {
		errlist = append(errlist, errors.New("Header.FileName contains embedded '\\' byte"))
	}
	return errlist
}

func checkHeaderComment(header Header, errlist []error) []error {
	if len(header.Comment) >= 256 {
		errlist = append(errlist, errors.New("Header.Comment is longer than 256 bytes"))
	}
	if index := strings.IndexByte(header.Comment, 0x00); index >= 0 {
		errlist = append(errlist, errors.New("Header.Comment contains embedded NUL byte"))
	}
	return errlist
}

func checkHeaderLastModified(header Header, errlist []error) []error {
	if !header.LastModified.IsZero() {
		s64 := header.LastModified.Unix()
		if s64 < 0 || (s64 >= 0 && uint64(s64) >= uint64(math.MaxUint32)) {
			errlist = append(errlist, errors.New("Header.LastModified is out of range for unsigned 32-bit time_t"))
		}
	}
	return errlist
}

func checkHeaderOSType(header Header, errlist []error) []error {
	if !header.OSType.IsValid() {
		errlist = append(errlist, errors.New("Header.OSType is not valid"))
	}
	return errlist
}

func checkHeaderExtraData(header Header, errlist []error) []error {
	for index, rec := range header.ExtraData.Records {
		if recLen := 4 + uint(len(rec.Bytes)); recLen > uint(math.MaxUint16) {
			errlist = append(errlist, fmt.Errorf("Header.ExtraData.Records[%d] encodes to %d bytes, which is beyond uint16_t", index, recLen))
		}
	}
	if xdata := header.ExtraData.AsBytes(); uint(len(xdata)) > uint(math.MaxUint16) {
		errlist = append(errlist, fmt.Errorf("Header.ExtraData encodes to %d bytes, which is beyond uint16_t", uint(len(xdata))))
	}
	return errlist
}

// Write writes a slice of bytes to the compressed stream.
// Conforms to the io.Writer interface.
func (fw *Writer) Write(buf []byte) (int, error) {
	length := uint(len(buf))

	fw.mu.Lock()
	defer fw.mu.Unlock()

	if fw.state == closedWriterState {
		return 0, fs.ErrClosed
	}

	if fw.state == errorWriterState {
		return 0, fw.err
	}

	var i, j uint
	for i < length {
		nn, err := fw.hybrid.Write(buf[i:])
		j = i + uint(nn)

		consumeAll := (nn == 0) || (err == buffer.ErrFull)
		if !fw.compressImpl(consumeAll) {
			return 0, fw.err
		}

		i = j
	}
	return int(length), nil
}

// Flush flushes the buffered compressed data to the underlying io.Writer.
func (fw *Writer) Flush(flushType FlushType) error {
	assert.Assertf(flushType.IsValid(), "invalid FlushType %d", uint(flushType))

	fw.mu.Lock()
	defer fw.mu.Unlock()

	if fw.state == closedWriterState {
		return fs.ErrClosed
	}

	if fw.state == errorWriterState {
		return fw.err
	}

	if !fw.flushImpl(flushType) {
		return fw.err
	}

	return nil
}

// Close finishes the compressed stream and closes this Writer.
//
// The underlying io.Writer is *not* closed, even if it supports io.Closer.
//
// The only method which is guaranteed to be safe to call on a Writer after
// Close is Reset, which will return the Writer to a non-closed state.
//
func (fw *Writer) Close() error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if fw.state == closedWriterState {
		return fs.ErrClosed
	}

	helper := func() error {
		err := fw.err
		fw.state = closedWriterState
		fw.err = nil
		return err
	}

	if fw.state == errorWriterState {
		return helper()
	}

	if !fw.flushImpl(FinishFlush) {
		return helper()
	}

	fw.state = closedWriterState
	return nil
}

func (fw *Writer) compressImpl(consumeAll bool) bool {
	if fw.hybrid.IsEmpty() {
		return true
	}

	if fw.state == closingStreamWriterState {
		if !fw.writeFooter() {
			return false
		}
		fw.state = closedStreamWriterState
	}

	if fw.state == noStreamWriterState || fw.state == closedStreamWriterState {
		if !fw.writeHeader() {
			return false
		}
		fw.state = openStreamWriterState
	}

	if !fw.cfg.compressFn(fw, consumeAll, fw.cfg.good, fw.cfg.lazy, fw.cfg.nice, fw.cfg.chain) {
		return false
	}

	return true
}

func (fw *Writer) flushImpl(flushType FlushType) bool {
	if !fw.compressImpl(true) {
		return false
	}

	if flushType == FinishFlush && fw.state == noStreamWriterState {
		if !fw.writeHeader() {
			return false
		}
		fw.state = openStreamWriterState
	}

	if fw.state == openStreamWriterState {
		if !fw.flushGuts(flushType) {
			return false
		}
	}

	if fw.state == closingStreamWriterState {
		if !fw.writeFooter() {
			return false
		}

		fw.state = closedStreamWriterState
	}

	if x, ok := fw.w.(flushWriter); ok {
		if err := x.Flush(); err != nil {
			fw.state = errorWriterState
			fw.err = err
			return false
		}
	}

	if x, ok := fw.w.(syncWriter); ok {
		if err := x.Sync(); err != nil && !isIgnoredSyncError(err) {
			fw.state = errorWriterState
			fw.err = err
			return false
		}
	}

	return true
}

func (fw *Writer) flushGuts(flushType FlushType) bool {
	switch flushType {
	case BlockFlush:
		// pass

	case PartialFlush:
		tokens := []token{makeStopToken()}
		if !writeStaticBlock(fw, tokens, false) {
			return false
		}

	case SyncFlush:
		if !writeStoredBlock(fw, nil, false) {
			return false
		}

	case FullFlush:
		if !writeStoredBlock(fw, nil, false) {
			return false
		}
		fw.hybrid.WindowClear()

	case FinishFlush:
		if !writeStoredBlock(fw, nil, true) {
			return false
		}
		fw.hybrid.WindowClear()
		fw.state = closingStreamWriterState
	}
	return true
}

func (fw *Writer) writeHeader() bool {
	fw.inputBytesStream = 0
	fw.outputBytesStream = 0
	fw.numStreams++

	fw.hybrid.WindowClear()
	if fw.dict != nil {
		fw.hybrid.SetWindow(fw.dict)
	}

	fw.sendEvent(Event{
		Type: StreamBeginEvent,
	})

	if fw.header.DataType == UnknownData {
		fw.header.DataType = BinaryData
		if buf := fw.hybrid.PrepareBulkRead(4096); isText(buf) {
			fw.header.DataType = TextData
		}
	}

	if fw.header.OSType == OSTypeUnknown {
		fw.header.OSType = OSTypeUnix
	}

	fw.header.WindowBits = fw.wbits
	fw.header.CompressLevel = fw.clevel

	var ok bool
	switch fw.format {
	case RawFormat:
		ok = true
	case ZlibFormat:
		ok = fw.writeHeaderZlib()
	case GZIPFormat:
		ok = fw.writeHeaderGZIP()
	default:
		assert.Raisef("Format %#v not implemented", fw.format)
	}

	if !ok {
		return false
	}

	h := new(Header)
	*h = fw.header
	fw.sendEvent(Event{
		Type:   StreamHeaderEvent,
		Header: h,
	})

	fw.inputBytesAdler32.Reset()
	fw.inputBytesCRC32.Reset()
	return true
}

func (fw *Writer) writeFooter() bool {
	if !fw.outputBitsFlush() {
		return false
	}

	a32 := fw.inputBytesAdler32.Sum32()
	c32 := fw.inputBytesCRC32.Sum32()

	fw.sendEvent(Event{
		Type: StreamEndEvent,
		Footer: &FooterEvent{
			Adler32: Checksum32(a32),
			CRC32:   Checksum32(c32),
		},
	})

	var ok bool
	switch fw.format {
	case RawFormat:
		ok = true
	case ZlibFormat:
		ok = fw.writeFooterZlib(a32)
	case GZIPFormat:
		ok = fw.writeFooterGZIP(c32)
	default:
		assert.Raisef("Format %#v not implemented", fw.format)
	}

	if !ok {
		return false
	}

	fw.sendEvent(Event{
		Type: StreamCloseEvent,
	})

	return true
}

func (fw *Writer) writeHeaderZlib() bool {
	var storage [6]byte
	var n uint = 2

	var fLevel byte
	if fw.strategy == HuffmanOnlyStrategy || fw.strategy == HuffmanRLEStrategy || fw.clevel < 2 {
		fLevel = 0x00
	} else if fw.clevel < 6 {
		fLevel = 0x01
	} else if fw.clevel == 6 {
		fLevel = 0x02
	} else {
		fLevel = 0x03
	}

	storage[0] = ((byte(fw.wbits-8) << 4) | 0x08)
	storage[1] = (fLevel << 6)

	if fw.dict != nil {
		binary.BigEndian.PutUint32(storage[2:6], adler32.Checksum(fw.dict))
		storage[1] |= 0x20
		n = 6
	}

	// compute header checksum
	u16 := binary.BigEndian.Uint16(storage[0:2])
	remainder := (u16 % 31)
	if remainder == 0 {
		remainder = 31
	}
	storage[1] |= byte(31 - remainder)

	return fw.outputBufferWrite(storage[0:n])
}

func (fw *Writer) writeFooterZlib(a32 uint32) bool {
	var storage [4]byte
	binary.BigEndian.PutUint32(storage[0:4], a32)
	return fw.outputBufferWrite(storage[0:4])
}

func (fw *Writer) writeHeaderGZIP() bool {
	fw.outputBytesCRC32.Reset()

	var header [10]byte

	header[0] = 0x1f // ID1
	header[1] = 0x8b // ID2
	header[2] = 0x08 // CM  (0x08 = DEFLATE)
	header[3] = 0x02 // FLG (0x02 = FHCRC)
	header[4] = 0x00 // MTIME[0]
	header[5] = 0x00 // MTIME[1]
	header[6] = 0x00 // MTIME[2]
	header[7] = 0x00 // MTIME[3]
	header[8] = 0x00 // XFL (0x00 = other)
	header[9] = 0xff // OS  (0xff = unknown)

	bits0 := writeHeaderGZIPDataType(fw.header)
	bits1, func1 := writeHeaderGZIPExtraData(fw.header)
	bits2, func2 := writeHeaderGZIPFileName(fw.header)
	bits3, func3 := writeHeaderGZIPComment(fw.header)

	header[3] |= (bits0 | bits1 | bits2 | bits3)

	if !fw.header.LastModified.IsZero() {
		s64 := fw.header.LastModified.Unix()
		u32 := uint32(uint64(s64))
		binary.LittleEndian.PutUint32(header[4:8], u32)
	}

	if fw.strategy == HuffmanOnlyStrategy || fw.strategy == HuffmanRLEStrategy || fw.clevel < 2 {
		header[8] = 0x04
	} else if fw.clevel == BestCompression {
		header[8] = 0x02
	}

	if osByte, found := gzipOSTypeEncodeTable[fw.header.OSType]; found {
		header[9] = osByte
	}

	ok := true
	ok = ok && fw.outputBufferWrite(header[:])
	ok = ok && func1(fw)
	ok = ok && func2(fw)
	ok = ok && func3(fw)

	c32 := fw.outputBytesCRC32.Sum32()

	ok = ok && fw.outputBufferWriteU16(binary.LittleEndian, uint16(c32))
	return ok
}

func writeHeaderGZIPDataType(header Header) byte {
	if header.DataType != TextData {
		return 0x00
	}
	// 0x01 = FTEXT
	return 0x01
}

func writeHeaderGZIPExtraData(header Header) (byte, func(*Writer) bool) {
	if len(header.ExtraData.Records) == 0 {
		return 0x00, func(*Writer) bool { return true }
	}

	// 0x04 = FEXTRA
	return 0x04, func(fw *Writer) bool {
		xdata := fw.header.ExtraData.AsBytes()
		xlen := uint16(len(xdata))
		ok := fw.outputBufferWriteU16(binary.LittleEndian, xlen)
		ok = ok && fw.outputBufferWrite(xdata)
		return ok
	}
}

func writeHeaderGZIPFileName(header Header) (byte, func(*Writer) bool) {
	if header.FileName == "" {
		return 0x00, func(*Writer) bool { return true }
	}

	// 0x08 = FNAME
	return 0x08, func(fw *Writer) bool {
		return fw.outputBufferWriteStringZ(header.FileName)
	}
}

func writeHeaderGZIPComment(header Header) (byte, func(*Writer) bool) {
	if header.Comment == "" {
		return 0x00, func(*Writer) bool { return true }
	}

	// 0x10 = FCOMMENT
	return 0x10, func(fw *Writer) bool {
		return fw.outputBufferWriteStringZ(header.Comment)
	}
}

func (fw *Writer) writeFooterGZIP(c32 uint32) bool {
	ok1 := fw.outputBufferWriteU32(binary.LittleEndian, c32)
	ok2 := fw.outputBufferWriteU32(binary.LittleEndian, uint32(fw.inputBytesStream))
	return ok1 && ok2
}

func (fw *Writer) inputBufferRead(buf []byte) []byte {
	nn, _ := fw.hybrid.Read(buf)
	buf = buf[:nn]
	fw.inputBytesAdler32.Write(buf)
	fw.inputBytesCRC32.Write(buf)
	fw.inputBytesTotal += uint64(len(buf))
	fw.inputBytesStream += uint64(len(buf))
	return buf
}

func (fw *Writer) inputBufferAdvance() (buf []byte, matchDistance uint, matchLength uint, matchFound bool) {
	buf, matchDistance, matchLength, matchFound = fw.hybrid.Advance()
	fw.inputBytesAdler32.Write(buf)
	fw.inputBytesCRC32.Write(buf)
	fw.inputBytesTotal += uint64(len(buf))
	fw.inputBytesStream += uint64(len(buf))
	return
}

func (fw *Writer) outputBufferWrite(buf []byte) bool {
	if !fw.outputBitsFlush() {
		return false
	}

	length := uint(len(buf))
	i := uint(0)
	for i < length {
		nn, _ := fw.output.Write(buf[i:])
		j := i + uint(nn)
		fw.outputBytesCRC32.Write(buf[i:j])
		fw.outputBytesTotal += uint64(nn)
		fw.outputBytesStream += uint64(nn)
		i = j
		if !fw.outputBufferFlush() {
			return false
		}
	}
	return true
}

func (fw *Writer) outputBufferWriteU16(bo binary.ByteOrder, u16 uint16) bool {
	var tmp [2]byte
	bo.PutUint16(tmp[:], u16)
	return fw.outputBufferWrite(tmp[:])
}

func (fw *Writer) outputBufferWriteU32(bo binary.ByteOrder, u32 uint32) bool {
	var tmp [4]byte
	bo.PutUint32(tmp[:], u32)
	return fw.outputBufferWrite(tmp[:])
}

func (fw *Writer) outputBufferWriteStringZ(str string) bool {
	strLen := len(str)
	last := strLen - 1
	index := strings.IndexByte(str, 0)
	var tmp []byte
	if index < 0 {
		tmp = make([]byte, strLen+1)
		copy(tmp, str)
		tmp[strLen] = 0
	} else if index == last {
		tmp = make([]byte, strLen)
		copy(tmp, str)
		tmp[last] = 0
	} else {
		assert.Raisef("string %q contains embedded ASCII NUL byte", str)
	}
	return fw.outputBufferWrite(tmp)
}

func (fw *Writer) outputBufferFlush() bool {
	_, err := fw.output.WriteTo(fw.w)
	if err != nil {
		fw.state = errorWriterState
		fw.err = err
		return false
	}
	return true
}

func (fw *Writer) outputBitsWrite(inLen byte, inBlock block) bool {
	assert.Assertf(fw.obLen <= bitsPerBlock, "obLen %d > bitsPerBlock %d", fw.obLen, bitsPerBlock)
	assert.Assertf(inLen <= bitsPerBlock, "inLen %d > bitsPerBlock %d", inLen, bitsPerBlock)

	for inLen != 0 {
		shiftA := fw.obLen
		shiftB := byte(bitsPerBlock - shiftA)
		if shiftB > inLen {
			shiftB = inLen
		}
		maskB := makeMask(shiftB)
		fw.obBlock = fw.obBlock | ((inBlock & maskB) << shiftA)
		inBlock = (inBlock >> shiftB)
		fw.obLen += shiftB
		inLen -= shiftB

		if fw.obLen == bitsPerBlock {
			if !fw.outputBitsFlush() {
				return false
			}
		}
	}
	return true
}

func (fw *Writer) outputBitsWriteHC(hc huffman.Code) bool {
	return fw.outputBitsWrite(hc.Size, block(hc.Bits))
}

func (fw *Writer) outputBitsFlush() bool {
	if fw.obLen == 0 {
		return true
	}

	var tmp [bytesPerBlock]byte
	n := ((fw.obLen + 7) / 8)
	bytesFromBlock(binary.LittleEndian, tmp[:], fw.obBlock)
	fw.obLen = 0
	fw.obBlock = 0
	return fw.outputBufferWrite(tmp[:n])
}

func (fw *Writer) sendEvent(event Event) {
	event.InputBytesTotal = fw.inputBytesTotal
	event.InputBytesStream = fw.inputBytesStream
	event.OutputBytesTotal = fw.outputBytesTotal
	event.OutputBytesStream = fw.outputBytesStream
	event.NumStreams = fw.numStreams
	event.Format = fw.format
	for _, tr := range fw.tracers {
		tr.OnEvent(event)
	}
}

func compressStore(fw *Writer, consumeAll bool, good, lazy, nice, chain uint) bool {
	const storedBlockSize = (1 << 16) - (1 << 12) // 64 KiB - 4 KiB

	available := fw.hybrid.Len()
	if available < storedBlockSize && !consumeAll && !fw.hybrid.IsFull() {
		return true
	}

	var block [storedBlockSize]byte

	for available != 0 {
		buf := fw.inputBufferRead(block[:])

		if !writeStoredBlock(fw, buf, false) {
			return false
		}

		available -= uint(len(buf))
	}
	return true
}

func compressHuff(fw *Writer, consumeAll bool, good, lazy, nice, chain uint) bool {
	return compressSlow(fw, consumeAll, good, lazy, nice, chain)
}

func compressRLE(fw *Writer, consumeAll bool, good, lazy, nice, chain uint) bool {
	return compressSlow(fw, consumeAll, good, lazy, nice, chain)
}

func compressFast(fw *Writer, consumeAll bool, good, lazy, nice, chain uint) bool {
	return compressSlow(fw, consumeAll, good, lazy, nice, chain)
}

func compressSlow(fw *Writer, consumeAll bool, good, lazy, nice, chain uint) bool {
	const compressedBlockSize = 4096

	available := fw.hybrid.Len()
	if available < compressedBlockSize && !consumeAll && !fw.hybrid.IsFull() {
		return true
	}

	bb := takeBytesBuffer()
	defer giveBytesBuffer(bb)
	bb.Grow(int(available))

	tokens := takeTokens()
	defer giveTokens(tokens)

	for available != 0 {
		p, distance, length, found := fw.inputBufferAdvance()
		if found {
			assert.Assertf(length >= 3, "length %d < 3", length)
			assert.Assertf(distance >= 1, "distance %d < 1", distance)
			bb.Write(p)
			*tokens = append(*tokens, makeCopyToken(uint16(length), uint16(distance)))
		} else {
			ch := p[0]
			bb.WriteByte(ch)
			*tokens = append(*tokens, makeLiteralToken(ch))
		}
		available -= uint(len(p))
	}

	data := bb.Bytes()
	*tokens = append(*tokens, makeStopToken())
	return compressTokens(fw, data, *tokens, false)
}

func compressTokens(bw bitwriter, data []byte, tokens []token, isFinal bool) bool {
	freqLL, freqD := studyFrequenciesLLD(tokens)

	var hLL huffman.Encoder
	hLL.Init(physicalNumLLCodes, freqLL)

	var hD huffman.Encoder
	hD.Init(physicalNumDCodes, freqD)

	var dynamicCounter bitcounter
	writeDynamicBlock(&dynamicCounter, &hLL, &hD, tokens, isFinal)

	var staticCounter bitcounter
	writeStaticBlock(&staticCounter, tokens, isFinal)

	storedLength := (uint64(len(data)) << 3) + 35
	if storedLength < dynamicCounter.length() {
		return writeStoredBlock(bw, data, isFinal)
	}
	if staticCounter.length() <= dynamicCounter.length() {
		return writeStaticBlock(bw, tokens, isFinal)
	}
	return writeDynamicBlock(bw, &hLL, &hD, tokens, isFinal)
}

func writeStoredBlock(bw bitwriter, data []byte, isFinal bool) bool {
	assert.Assertf(uint(len(data)) <= uint(math.MaxUint16), "uncompressed block length %d exceeds uint16_t", uint(len(data)))

	bw.sendEvent(Event{
		Type: BlockBeginEvent,
		Block: &BlockEvent{
			Type:    StoredBlock,
			IsFinal: isFinal,
		},
	})

	// BTYPE=00 BFINAL=x
	bits := block(0x00)
	if isFinal {
		bits = block(0x01)
	}

	u16 := uint16(len(data))

	if !bw.outputBitsWrite(3, bits) {
		return false
	}
	if !bw.outputBitsFlush() {
		return false
	}
	if !bw.outputBufferWriteU16(binary.LittleEndian, u16) {
		return false
	}
	if !bw.outputBufferWriteU16(binary.LittleEndian, ^u16) {
		return false
	}
	if !bw.outputBufferWrite(data) {
		return false
	}

	bw.sendEvent(Event{
		Type: BlockEndEvent,
		Block: &BlockEvent{
			Type:    StoredBlock,
			IsFinal: isFinal,
		},
	})

	return true
}

func writeStaticBlock(bw bitwriter, tokens []token, isFinal bool) bool {
	tokensLen := uint(len(tokens))
	assert.Assert(tokensLen >= 1, "at least 1 token is required")
	assert.Assert(tokens[tokensLen-1].tokenType() == stopToken, "last token must be a stop token")

	hLL, hD := getFixedHuffEncoders()
	sLL := hLL.SizeBySymbol()
	sD := hD.SizeBySymbol()

	bw.sendEvent(Event{
		Type: BlockBeginEvent,
		Block: &BlockEvent{
			Type:    StaticBlock,
			IsFinal: isFinal,
		},
	})

	// BTYPE=01 BFINAL=x
	bits := block(0x02)
	if isFinal {
		bits = block(0x03)
	}

	if !bw.outputBitsWrite(3, bits) {
		return false
	}

	bw.sendEvent(Event{
		Type: BlockTreesEvent,
		Block: &BlockEvent{
			Type:    StaticBlock,
			IsFinal: isFinal,
		},
		Trees: &TreesEvent{
			LiteralLengthSizes: sLL,
			DistanceSizes:      sD,
		},
	})

	for _, t := range tokens {
		if !t.encodeLLD(bw, hLL, hD) {
			return false
		}
	}

	bw.sendEvent(Event{
		Type: BlockEndEvent,
		Block: &BlockEvent{
			Type:    StaticBlock,
			IsFinal: isFinal,
		},
	})

	return true
}

func writeDynamicBlock(bw bitwriter, hLL *huffman.Encoder, hD *huffman.Encoder, tokens []token, isFinal bool) bool {
	tokensLen := uint(len(tokens))
	assert.Assert(tokensLen >= 1, "at least 1 token is required")
	assert.Assert(tokens[tokensLen-1].tokenType() == stopToken, "last token must be a stop token")

	sLL := hLL.SizeBySymbol()
	sD := hD.SizeBySymbol()

	bw.sendEvent(Event{
		Type: BlockBeginEvent,
		Block: &BlockEvent{
			Type:    DynamicBlock,
			IsFinal: isFinal,
		},
	})

	// BTYPE=10 BFINAL=x
	bits := block(0x04)
	if isFinal {
		bits = block(0x05)
	}

	if !bw.outputBitsWrite(3, bits) {
		return false
	}

	xtokens := takeTokens()
	defer giveTokens(xtokens)

	var numLL, numD uint
	*xtokens, numLL = encodeTreeTokens(*xtokens, sLL, 257)
	*xtokens, numD = encodeTreeTokens(*xtokens, sD, 1)

	freqX := studyFrequenciesX(*xtokens)

	var hX huffman.Encoder
	hX.Init(physicalNumXCodes, freqX)

	sX := hX.SizeBySymbol()
	numX := uint(physicalNumXCodes)
	for numX > 4 && sX[scramble[numX-1]] == 0 {
		numX--
	}

	if !bw.outputBitsWrite(5, block(numLL-257)) {
		return false
	}
	if !bw.outputBitsWrite(5, block(numD-1)) {
		return false
	}
	if !bw.outputBitsWrite(4, block(numX-4)) {
		return false
	}
	for i := uint(0); i < numX; i++ {
		if !bw.outputBitsWrite(3, block(sX[scramble[i]])) {
			return false
		}
	}
	for _, t := range *xtokens {
		if !t.encodeX(bw, &hX) {
			return false
		}
	}

	bw.sendEvent(Event{
		Type: BlockTreesEvent,
		Block: &BlockEvent{
			Type:    DynamicBlock,
			IsFinal: isFinal,
		},
		Trees: &TreesEvent{
			CodeCount:          uint16(numX),
			LiteralLengthCount: uint16(numLL),
			DistanceCount:      uint16(numD),
			CodeSizes:          sX,
			LiteralLengthSizes: sLL,
			DistanceSizes:      sD,
		},
	})

	for _, t := range tokens {
		if !t.encodeLLD(bw, hLL, hD) {
			return false
		}
	}

	bw.sendEvent(Event{
		Type: BlockEndEvent,
		Block: &BlockEvent{
			Type:    DynamicBlock,
			IsFinal: isFinal,
		},
	})

	return true
}

type compressFunc func(fw *Writer, consumeAll bool, good, lazy, nice, chain uint) bool

type compressConfig struct {
	good       uint
	lazy       uint
	nice       uint
	chain      uint
	compressFn compressFunc
}

var compressConfigs = [...]compressConfig{
	{0x0000, 0x0000, 0x0000, 0x0000, compressRLE},
	{0x0000, 0x0000, 0x0000, 0x0000, compressHuff},
	{0x0000, 0x0000, 0x0000, 0x0000, compressStore},
	{0x0004, 0x0004, 0x0008, 0x0004, compressFast},
	{0x0004, 0x0005, 0x0010, 0x0008, compressFast},
	{0x0004, 0x0006, 0x0020, 0x0020, compressFast},
	{0x0004, 0x0004, 0x0010, 0x0010, compressSlow},
	{0x0008, 0x0010, 0x0020, 0x0020, compressSlow},
	{0x0008, 0x0010, 0x0080, 0x0080, compressSlow},
	{0x0008, 0x0020, 0x0080, 0x0100, compressSlow},
	{0x0020, 0x0080, 0x0102, 0x0400, compressSlow},
	{0x0020, 0x0102, 0x0102, 0x1000, compressSlow},
}

func isIgnoredSyncError(err error) bool {
	return errors.Is(err, syscall.EINVAL)
}
