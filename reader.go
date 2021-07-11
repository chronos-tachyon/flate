package flate

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/chronos-tachyon/assert"
	buffer "github.com/chronos-tachyon/buffer/v3"
	"github.com/chronos-tachyon/bufferpool"
	"github.com/chronos-tachyon/huffman"

	"github.com/chronos-tachyon/flate/internal/adler32"
	"github.com/chronos-tachyon/flate/internal/crc32"
)

// Reader wraps an io.Reader and decompresses the data which flows through it.
type Reader struct {
	// wg keeps track of any spawned background threads.
	wg sync.WaitGroup

	// mu1 protects most Reader fields
	mu1 sync.Mutex

	// mu2 protects outputError & outputBuffer
	mu2 sync.Mutex

	// cv1 announces "please drain outputBuffer".
	// - Wait'ed by Read callers to block until outputBuffer is non-empty.
	// - Signal'ed by Read callers when outputBuffer still has more data after reading.
	// - Signal'ed by readThread when outputBuffer is filled.
	// - Broadcast'ed by readThread when outputError is set.
	cv1 *sync.Cond

	// cv2 announces "please fill outputBuffer".
	// - Wait'ed by readThread to block until outputBuffer is non-full (ideally empty).
	// - Signal'ed by Read callers when outputBuffer is empty.
	cv2 *sync.Cond

	// fields below are protected by mu1

	format  Format
	mlevel  MemoryLevel
	wbits   WindowBits
	dict    []byte
	tracers []Tracer

	r                    io.Reader
	inputError           error
	inputBuffer          buffer.Buffer
	decodeBuffer         buffer.Buffer
	window               buffer.Window
	inputBytesTotal      uint64
	inputBytesStream     uint64
	outputBytesTotal     uint64
	outputBytesStream    uint64
	numStreams           uint
	inputBytesCRC32      crc32.Hash
	outputBytesCRC32     crc32.Hash
	outputBytesAdler32   adler32.Hash
	ibBlock              block
	ibLen                byte
	readThreadWasStarted bool
	readThreadWasStopped bool
	outputErrorWasSet    bool
	stopIsForced         bool

	header       Header
	actualFormat Format

	hLL *huffman.Decoder
	hD  *huffman.Decoder

	hd0 huffman.Decoder
	hd1 huffman.Decoder
	hd2 huffman.Decoder

	tmp [4]byte

	// fields below are protected by mu2

	outputError  error
	outputBuffer buffer.Buffer
}

// NewReader constructs and returns a new Reader with the given io.Reader and
// options.
func NewReader(r io.Reader, opts ...Option) *Reader {
	assert.NotNil(&r)

	var o options
	o.reset()
	o.apply(opts)
	o.populateReaderDefaults()

	fr := &Reader{
		format:  o.format,
		mlevel:  o.mlevel,
		wbits:   o.wbits,
		dict:    o.dict,
		tracers: o.tracers,

		r: r,
	}

	fr.cv1 = sync.NewCond(&fr.mu2)
	fr.cv2 = sync.NewCond(&fr.mu2)

	fr.inputBuffer.Init(fr.inputBufferNumBits())
	fr.decodeBuffer.Init(fr.decodeBufferNumBits())
	fr.window.Init(fr.windowNumBits())

	fr.outputBuffer.Init(fr.outputBufferNumBits())

	return fr
}

func (fr *Reader) inputBufferNumBits() uint {
	return uint(fr.mlevel + 6)
}

func (fr *Reader) decodeBufferNumBits() uint {
	return uint(fr.mlevel + 6)
}

func (fr *Reader) outputBufferNumBits() uint {
	return uint(fr.mlevel + 6)
}

func (fr *Reader) windowNumBits() uint {
	return uint(fr.wbits)
}

// Format returns the Format which this Reader uses.
func (fr *Reader) Format() Format {
	fr.mu1.Lock()
	format := fr.format
	fr.mu1.Unlock()
	return format
}

// MemoryLevel returns the MemoryLevel which this Reader uses.
func (fr *Reader) MemoryLevel() MemoryLevel {
	fr.mu1.Lock()
	mlevel := fr.mlevel
	fr.mu1.Unlock()
	return mlevel
}

// WindowBits returns the WindowBits which this Reader uses.
func (fr *Reader) WindowBits() WindowBits {
	fr.mu1.Lock()
	wbits := fr.wbits
	fr.mu1.Unlock()
	return wbits
}

// Dict returns the pre-shared LZ77 dictionary which this Reader uses, or nil
// if no such dictionary is in use.
func (fr *Reader) Dict() []byte {
	var dict []byte
	fr.mu1.Lock()
	if len(fr.dict) != 0 {
		dict = make([]byte, len(fr.dict))
		copy(dict, fr.dict)
	}
	fr.mu1.Unlock()
	return dict
}

// Tracers returns the Tracers which this Reader uses.
func (fr *Reader) Tracers() []Tracer {
	var tracers []Tracer
	fr.mu1.Lock()
	if len(fr.tracers) != 0 {
		tracers = make([]Tracer, len(fr.tracers))
		copy(tracers, fr.tracers)
	}
	fr.mu1.Unlock()
	return tracers
}

// UnderlyingReader returns the io.Reader which this Reader uses.
func (fr *Reader) UnderlyingReader() io.Reader {
	return fr.r
}

// Reset re-initializes this Reader with the given io.Reader and options.  Any
// options given here are merged with all previous options.
func (fr *Reader) Reset(r io.Reader, opts ...Option) {
	assert.NotNil(&r)
	for _, opt := range opts {
		assert.NotNil(&opt)
	}

	fr.mu1.Lock()
	defer fr.mu1.Unlock()

	fr.mu2.Lock()
	defer fr.mu2.Unlock()

	fr.stopReadThreadLocked()

	fr.r = r
	fr.inputError = nil
	fr.readThreadWasStarted = false
	fr.readThreadWasStopped = false
	fr.outputErrorWasSet = false
	fr.stopIsForced = false

	fr.inputBuffer.Clear()
	fr.decodeBuffer.Clear()
	fr.window.Clear()

	fr.outputError = nil
	fr.outputBuffer.Clear()

	if len(opts) == 0 {
		return
	}

	var o options
	o.reset()
	o.format = fr.format
	o.mlevel = fr.mlevel
	o.wbits = fr.wbits
	o.dict = fr.dict
	o.tracers = fr.tracers
	o.apply(opts)
	o.populateReaderDefaults()

	fr.format = o.format
	fr.mlevel = o.mlevel
	fr.wbits = o.wbits
	fr.dict = o.dict
	fr.tracers = o.tracers

	if numBits := fr.inputBufferNumBits(); fr.inputBuffer.NumBits() != numBits {
		fr.inputBuffer.Init(numBits)
	}
	if numBits := fr.decodeBufferNumBits(); fr.decodeBuffer.NumBits() != numBits {
		fr.decodeBuffer.Init(numBits)
	}
	if numBits := fr.windowNumBits(); fr.window.NumBits() != numBits {
		fr.window.Init(numBits)
	}
	if numBits := fr.outputBufferNumBits(); fr.outputBuffer.NumBits() != numBits {
		fr.outputBuffer.Init(numBits)
	}
}

// Read reads from the compressed stream into the provided slice of bytes.
// Conforms to the io.Reader interface.
func (fr *Reader) Read(p []byte) (int, error) {
	fr.startReadThread()

	fr.mu2.Lock()
	defer fr.mu2.Unlock()

	if fr.outputError == nil && fr.outputBuffer.IsEmpty() {
		fr.cv2.Signal()
		for fr.outputError == nil && fr.outputBuffer.IsEmpty() {
			fr.cv1.Wait()
		}
	}

	if fr.outputBuffer.IsEmpty() {
		return 0, fr.outputError
	}

	nn, _ := fr.outputBuffer.Read(p)

	if fr.outputBuffer.IsEmpty() {
		fr.cv2.Signal()
	} else {
		fr.cv1.Signal()
	}

	return nn, nil
}

// Close terminates decompression and closes this Reader.
//
// The underlying io.Reader is *not* closed, even if it supports io.Closer.
//
// The only method which is guaranteed to be safe to call on a Reader after
// Close is Reset, which will return the Reader to a non-closed state.
//
func (fr *Reader) Close() error {
	fr.mu1.Lock()
	fr.stopReadThreadLocked()
	fr.mu1.Unlock()
	return nil
}

func (fr *Reader) startReadThread() {
	fr.mu1.Lock()
	if !fr.readThreadWasStarted {
		fr.readThreadWasStarted = true
		fr.wg.Add(1)
		go fr.readThread()
	}
	fr.mu1.Unlock()
}

func (fr *Reader) stopReadThreadLocked() {
	if fr.readThreadWasStarted && !fr.readThreadWasStopped {
		fr.mu2.Lock()
		fr.outputError = context.Canceled // FIXME
		fr.outputBuffer.Clear()
		fr.cv1.Broadcast()
		fr.cv2.Broadcast()
		fr.mu2.Unlock()

		fr.outputErrorWasSet = true
		fr.stopIsForced = true

		fr.mu1.Unlock()
		fr.wg.Wait()
		fr.mu1.Lock()
	}
}

func (fr *Reader) readThread() {
	fr.mu1.Lock()

	fr.numStreams = 0
	for fr.inputError == nil {
		if !fr.readHeader() {
			break
		}

		fr.outputBytesAdler32.Reset()
		fr.outputBytesCRC32.Reset()

		for fr.readFlateBlock() {
			// pass
		}

		fr.decodeBufferMustFlush()

		if !fr.readFooter() {
			break
		}
	}

	if fr.inputError == nil {
		fr.inputError = io.EOF
	}

	fr.propagateError(false)
	fr.readThreadWasStopped = true
	fr.mu1.Unlock()
	fr.wg.Done()
}

func (fr *Reader) readHeader() bool {
	fr.inputBytesStream = 0
	fr.outputBytesStream = 0

	if fr.inputBuffer.IsEmpty() {
		fr.inputBufferFill()
		if fr.inputBuffer.IsEmpty() {
			fr.propagateError(false)
			return false
		}
	}

	fr.numStreams++

	fr.window.Clear()
	if fr.dict != nil {
		_, _ = fr.window.Write(fr.dict)
	}

	fr.actualFormat = DefaultFormat

	fr.sendEvent(Event{
		Type: StreamBeginEvent,
	})

	fr.header = Header{}

	var ok bool
	switch fr.format {
	case DefaultFormat:
		ok = fr.readHeaderAuto()
	case RawFormat:
		ok = fr.readHeaderRaw()
	case ZlibFormat:
		ok = fr.readHeaderZlib()
	case GZIPFormat:
		ok = fr.readHeaderGZIP()
	default:
		assert.Raisef("Format %#v not implemented", fr.format)
	}

	if !ok {
		return false
	}

	h := new(Header)
	*h = fr.header
	fr.sendEvent(Event{
		Type:   StreamHeaderEvent,
		Header: h,
	})

	return true
}

func (fr *Reader) readFooter() bool {
	fr.inputBitsDiscard()

	a32 := fr.outputBytesAdler32.Sum32()
	c32 := fr.outputBytesCRC32.Sum32()

	fr.sendEvent(Event{
		Type: StreamEndEvent,
		Footer: &FooterEvent{
			Adler32: Checksum32(a32),
			CRC32:   Checksum32(c32),
		},
	})

	var ok bool
	switch fr.actualFormat {
	case RawFormat:
		ok = fr.readFooterRaw()
	case ZlibFormat:
		ok = fr.readFooterZlib(a32)
	case GZIPFormat:
		ok = fr.readFooterGZIP(c32)
	default:
		assert.Raisef("Format %#v not implemented", fr.format)
	}

	if !ok {
		return false
	}

	fr.sendEvent(Event{
		Type: StreamCloseEvent,
	})

	return true
}

func (fr *Reader) readHeaderAuto() bool {
	p := fr.inputBuffer.PrepareBulkRead(2)
	if len(p) < 2 {
		return fr.readHeaderRaw()
	}

	if p[0] == 0x1f && p[1] == 0x8b {
		return fr.readHeaderGZIP()
	}

	u16 := binary.BigEndian.Uint16(p)
	if (p[0]&0x0f) == 0x08 && (u16%31) == 0 {
		return fr.readHeaderZlib()
	}

	return fr.readHeaderRaw()
}

func (fr *Reader) readHeaderRaw() bool {
	fr.actualFormat = RawFormat
	fr.header.CompressLevel = DefaultCompression
	return true
}

func (fr *Reader) readFooterRaw() bool {
	return true
}

func (fr *Reader) readHeaderZlib() bool {
	var header [2]byte
	p, ok := fr.inputBufferRead(header[:])
	if !ok {
		fr.propagateError(true)
		return false
	}

	u16 := binary.BigEndian.Uint16(p)
	if mod := (u16 % 31); mod != 0 {
		fr.corruptf("invalid zlib header checksum -- expected %#04x mod 31 == 0, got %d", u16, mod)
		return false
	}

	method := (p[0] & 0x0f)
	if method != 0x08 {
		fr.corruptf("invalid zlib compression method -- expected 0x8 (DEFLATE), got %#x", method)
		return false
	}

	fr.header.WindowBits = 8 + WindowBits(p[0]>>4)
	if fr.header.WindowBits > fr.wbits {
		fr.corruptf("zlib window size is too big -- data uses 2**%d, but this Reader is limited to 2**%d", fr.header.WindowBits, fr.wbits)
		return false
	}

	fr.header.CompressLevel = [4]CompressLevel{1, 2, DefaultCompression, 9}[p[1]>>6]

	bitFDICT := (p[1] & 0x20) != 0

	if bitFDICT {
		expectedAdler32, ok := fr.inputBufferReadU32(binary.BigEndian)
		if !ok {
			fr.propagateError(true)
			return false
		}

		if fr.window.IsZero() {
			fr.corruptf("zlib stream was compressed with a pre-set dictionary -- Adler-32 checksum of the dictionary required to decompress this stream is %#08x", expectedAdler32)
			return false
		}

		computedAdler32 := adler32.Checksum(fr.dict)
		if expectedAdler32 != computedAdler32 {
			fr.corruptf("zlib stream was compressed with a different pre-set dictionary -- Adler-32 checksum of the required dictionary is %#08x, checksum of the provided dictionary is %#08x", expectedAdler32, computedAdler32)
			return false
		}
	} else {
		if !fr.window.IsZero() {
			computedAdler32 := adler32.Checksum(fr.dict)
			fr.corruptf("zlib stream was not compressed with a pre-set dictionary -- Adler-32 checksum of the supplied dictionary is %#08x", computedAdler32)
			return false
		}
	}

	fr.actualFormat = ZlibFormat
	return true
}

func (fr *Reader) readFooterZlib(computedAdler32 uint32) bool {
	expectedAdler32, ok := fr.inputBufferReadU32(binary.BigEndian)
	if !ok {
		fr.propagateError(true)
		return false
	}

	if expectedAdler32 != computedAdler32 {
		fr.corruptf("invalid zlib Adler-32 checksum -- footer value %#08x, computed value %#08x", expectedAdler32, computedAdler32)
		return false
	}

	return true
}

func (fr *Reader) readHeaderGZIP() bool {
	fr.inputBytesCRC32.Reset()

	var header [10]byte
	p, ok := fr.inputBufferRead(header[:])
	if !ok {
		fr.propagateError(true)
		return false
	}

	if p[0] != 0x1f || p[1] != 0x8b {
		fr.corruptf("invalid gzip header identification bytes")
		return false
	}

	if p[2] != 0x08 {
		fr.corruptf("invalid gzip compression method %#02x -- expected 0x08 (DEFLATE)", p[2])
		return false
	}

	mtime := binary.LittleEndian.Uint32(p[4:8])
	if mtime != 0 {
		fr.header.LastModified = time.Unix(int64(mtime), 0)
	}

	fr.header.CompressLevel = DefaultCompression
	switch p[8] {
	case 0x02:
		fr.header.CompressLevel = 9
	case 0x04:
		fr.header.CompressLevel = 1
	}

	fr.header.OSType = gzipOSTypeDecodeTable[p[9]]

	bitFTEXT := (p[3] & 0x01) != 0
	bitFHCRC := (p[3] & 0x02) != 0
	bitFEXTRA := (p[3] & 0x04) != 0
	bitFNAME := (p[3] & 0x08) != 0
	bitFCOMMENT := (p[3] & 0x10) != 0
	if (p[3] & 0xe0) != 0 {
		fr.corruptf("invalid gzip flag bits %#02x", p[3]&0xe0)
		return false
	}

	fr.header.DataType = fr.readHeaderGZIPDataType(bitFTEXT)

	ok = ok && fr.readHeaderGZIPExtraData(bitFEXTRA, &fr.header)
	ok = ok && fr.readHeaderGZIPFileName(bitFNAME, &fr.header)
	ok = ok && fr.readHeaderGZIPComment(bitFCOMMENT, &fr.header)

	c32 := fr.inputBytesCRC32.Sum32()

	ok = ok && fr.readHeaderGZIPCRC16(bitFHCRC, c32)
	if !ok {
		return false
	}

	fr.actualFormat = GZIPFormat
	return true
}

func (fr *Reader) readHeaderGZIPDataType(bit bool) DataType {
	dataType := BinaryData
	if bit {
		dataType = TextData
	}
	return dataType
}

func (fr *Reader) readHeaderGZIPExtraData(bit bool, header *Header) bool {
	if bit {
		rawXLen, ok := fr.inputBufferReadU16(binary.LittleEndian)
		if !ok {
			fr.propagateError(true)
			return false
		}

		rawXData, ok := fr.inputBufferRead(make([]byte, rawXLen))
		if !ok {
			fr.propagateError(true)
			return false
		}

		header.ExtraData.Parse(rawXData)
	}
	return true
}

func (fr *Reader) readHeaderGZIPFileName(bit bool, header *Header) bool {
	if bit {
		str, ok := fr.inputBufferReadStringZ()
		if !ok {
			fr.propagateError(true)
			return false
		}

		header.FileName = str
	}
	return true
}

func (fr *Reader) readHeaderGZIPComment(bit bool, header *Header) bool {
	if bit {
		str, ok := fr.inputBufferReadStringZ()
		if !ok {
			fr.propagateError(true)
			return false
		}

		header.Comment = str
	}
	return true
}

func (fr *Reader) readHeaderGZIPCRC16(bit bool, c32 uint32) bool {
	if bit {
		expectedHeaderCRC16, ok := fr.inputBufferReadU16(binary.LittleEndian)
		if !ok {
			fr.propagateError(true)
			return false
		}

		computedHeaderCRC16 := uint16(c32)
		if computedHeaderCRC16 != expectedHeaderCRC16 {
			fr.corruptf("invalid gzip header CRC-16 checksum -- header value %#04x, computed value %#04x", expectedHeaderCRC16, computedHeaderCRC16)
			return false
		}
	}
	return true
}

func (fr *Reader) readFooterGZIP(computedCRC32 uint32) bool {
	expectedCRC32, ok := fr.inputBufferReadU32(binary.LittleEndian)
	if !ok {
		fr.propagateError(true)
		return false
	}

	contentLen, ok := fr.inputBufferReadU32(binary.LittleEndian)
	if !ok {
		fr.propagateError(true)
		return false
	}

	if expectedCRC32 != computedCRC32 {
		fr.corruptf("invalid gzip CRC-32 checksum -- footer value %#08x, computed value %#08x", expectedCRC32, computedCRC32)
		return false
	}

	if contentLen != uint32(fr.outputBytesStream) {
		fr.corruptf("invalid gzip decompressed length (mod 2**32) -- footer value %d, computed value %d", contentLen, uint32(fr.outputBytesStream))
		return false
	}

	return true
}

func (fr *Reader) readFlateBlock() (more bool) {
	if ok := fr.inputBitsFill(3); !ok {
		fr.propagateError(true)
		return false
	}

	out := fr.inputBitsRead(3)

	isOK := false
	isFinal := (out & 0x01) != 0
	blockType := BlockType(1+byte(out>>1)) & 0x03

	fr.sendEvent(Event{
		Type: BlockBeginEvent,
		Block: &BlockEvent{
			Type:    blockType,
			IsFinal: isFinal,
		},
	})

	switch blockType {
	case StoredBlock:
		isOK = fr.readFlateBlockStored()

	case StaticBlock:
		isOK = fr.readFlateBlockStaticHuffman(isFinal)

	case DynamicBlock:
		isOK = fr.readFlateBlockDynamicHuffman(isFinal)

	default:
		fr.corruptf("BTYPE 11 is reserved")
	}

	if isOK {
		fr.sendEvent(Event{
			Type: BlockEndEvent,
			Block: &BlockEvent{
				Type:    blockType,
				IsFinal: isFinal,
			},
		})
	}

	return isOK && !isFinal
}

func (fr *Reader) readFlateBlockStored() bool {
	len0, ok := fr.inputBufferReadU16(binary.LittleEndian)
	if !ok {
		fr.propagateError(true)
		return false
	}

	len1, ok := fr.inputBufferReadU16(binary.LittleEndian)
	if !ok {
		fr.propagateError(true)
		return false
	}

	if len1 != ^len0 {
		fr.corruptf("got LEN %#04x NLEN %#04x, expected NLEN %#04x", len0, len1, ^len0)
		return false
	}

	p, ok := fr.inputBufferRead(make([]byte, len0))
	if !ok {
		fr.propagateError(true)
		return false
	}

	fr.decodeBufferWrite(p)

	return true
}

func (fr *Reader) readFlateBlockStaticHuffman(isFinal bool) bool {
	fr.hLL, fr.hD = getFixedHuffDecoders()

	fr.sendEvent(Event{
		Type: BlockTreesEvent,
		Block: &BlockEvent{
			Type:    StaticBlock,
			IsFinal: isFinal,
		},
		Trees: &TreesEvent{
			LiteralLengthSizes: fr.hLL.SizeBySymbol(),
			DistanceSizes:      fr.hD.SizeBySymbol(),
		},
	})

	return fr.decodeHuffmanBlock(StaticBlock, isFinal)
}

func (fr *Reader) readFlateBlockDynamicHuffman(isFinal bool) bool {
	if !fr.readDynamicTrees(isFinal) {
		return false
	}
	return fr.decodeHuffmanBlock(DynamicBlock, isFinal)
}

func (fr *Reader) decodeHuffmanBlock(blockType BlockType, isFinal bool) bool {
	hLL := fr.hLL
	hD := fr.hD

	for {
		symbol, ok := fr.readSymbol(hLL)
		if !ok {
			fr.corruptf("degenerate literal/length Huffman code")
			return false
		}

		t, ok := fr.decodeLL(symbol)
		if !ok {
			return false
		}

		if t.distance == 0 {
			if t.literalOrLength >= 256 {
				return true
			}
			fr.decodeBufferWriteByte(byte(t.literalOrLength))
			continue
		}

		symbol, ok = fr.readSymbol(hD)
		if !ok {
			fr.corruptf("degenerate distance Huffman code")
			return false
		}

		t, ok = fr.decodeD(symbol, t.literalOrLength)
		if !ok {
			return false
		}

		length := uint(t.literalOrLength)
		distance := uint(t.distance)

		p, err := fr.window.LookupSlice(distance, length)
		if err != nil {
			fr.corruptf("distance %d > window.Size %d", distance, fr.window.Size())
			return false
		}
		fr.decodeBufferWrite(p)
		length -= uint(len(p))

		for length != 0 {
			ch, err := fr.window.LookupByte(distance)
			if err != nil {
				fr.corruptf("distance %d > window.Size %d", distance, fr.window.Size())
				return false
			}
			fr.decodeBufferWriteByte(ch)
			length--
		}
	}
}

func (fr *Reader) decodeLL(symbol huffman.Symbol) (token, bool) {
	var length uint32
	var additionalBits byte
	switch {
	case symbol < 256:
		ch := byte(symbol)
		return makeLiteralToken(ch), true

	case symbol == 256:
		return makeStopToken(), true

	case symbol < 265:
		length = uint32(symbol - 254)
		additionalBits = 0

	case symbol < 269:
		length = uint32(2*symbol - 519)
		additionalBits = 1

	case symbol < 273:
		length = uint32(4*symbol - 1057)
		additionalBits = 2

	case symbol < 277:
		length = uint32(8*symbol - 2149)
		additionalBits = 3

	case symbol < 281:
		length = uint32(16*symbol - 4365)
		additionalBits = 4

	case symbol < 285:
		length = uint32(32*symbol - 8861)
		additionalBits = 5

	case symbol == 285:
		length = 258
		additionalBits = 0

	default:
		fr.corruptf("invalid literal/length symbol %d", symbol)
		return makeInvalidToken(), false
	}

	if additionalBits != 0 {
		if ok := fr.inputBitsFill(additionalBits); !ok {
			fr.propagateError(true)
			return makeInvalidToken(), false
		}

		out := fr.inputBitsRead(additionalBits)
		length += uint32(out)
	}

	return makeCopyToken(uint16(length), 1), true
}

func (fr *Reader) decodeD(symbol huffman.Symbol, length uint16) (token, bool) {
	var distance uint32
	var additionalBits byte
	switch {
	case symbol < 4:
		distance = uint32(symbol + 1)
		additionalBits = 0

	case symbol < logicalNumDCodes:
		x0 := (byte(symbol-2) >> 1)
		x1 := uint32(1) << (x0 + 1)
		x2 := uint32(0)
		if (symbol & 0x01) != 0 {
			x2 = uint32(1) << x0
		}
		distance = x1 + x2 + 1
		additionalBits = x0

	default:
		fr.corruptf("invalid distance symbol %d", symbol)
		return makeInvalidToken(), false
	}

	if additionalBits != 0 {
		if ok := fr.inputBitsFill(additionalBits); !ok {
			fr.propagateError(true)
			return makeInvalidToken(), false
		}

		out := fr.inputBitsRead(additionalBits)
		distance += uint32(out)
	}

	return makeCopyToken(length, uint16(distance)), true
}

func (fr *Reader) readDynamicTrees(isFinal bool) bool {
	// https://www.rfc-editor.org/rfc/rfc1951.html — Section 3.2.7

	if ok := fr.inputBitsFill(14); !ok {
		fr.propagateError(true)
		return false
	}

	out := fr.inputBitsRead(14)
	numLL := 257 + uint(out&0x1f)
	numD := 1 + uint((out>>5)&0x1f)
	numX := 4 + uint((out>>10)&0x0f)

	if numLL > logicalNumLLCodes {
		fr.corruptf("HLIT %d > %d", numLL, logicalNumLLCodes)
	}
	if numD > logicalNumDCodes {
		fr.corruptf("HDIST %d > %d", numD, logicalNumDCodes)
	}

	const numCodes = 19
	sX := [numCodes]byte{}

	for i := uint(0); i < numX; i++ {
		if ok := fr.inputBitsFill(3); !ok {
			fr.propagateError(true)
			return false
		}

		out = fr.inputBitsRead(3)
		sX[scramble[i]] = byte(out)
	}

	if err := fr.hd0.Init(sX[:]); err != nil {
		fr.corruptf("failed to initialize bootstrap Huffman decoder: %v", err)
		return false
	}

	total := numLL + numD
	combinedLengths := make([]byte, total)
	i := uint(0)
	for i < total {
		sym, ok := fr.readSymbol(&fr.hd0)
		if !ok {
			fr.corruptf("degenerate bootstrap Huffman code")
			return false
		}

		i, ok = fr.decodeX(sym, combinedLengths, i, total)
		if !ok {
			return false
		}
	}

	sLL := make([]byte, physicalNumLLCodes)
	copy(sLL, combinedLengths[:numLL])

	sD := make([]byte, physicalNumDCodes)
	copy(sD, combinedLengths[numLL:])

	if err := fr.hd1.Init(sLL); err != nil {
		fr.corruptf("failed to initialize literal/length Huffman decoder: %v", err)
		return false
	}

	if err := fr.hd2.Init(sD); err != nil {
		fr.corruptf("failed to initialize distance Huffman decoder: %v", err)
		return false
	}

	fr.sendEvent(Event{
		Type: BlockTreesEvent,
		Block: &BlockEvent{
			Type:    DynamicBlock,
			IsFinal: isFinal,
		},
		Trees: &TreesEvent{
			CodeCount:          uint16(numX),
			LiteralLengthCount: uint16(numLL),
			DistanceCount:      uint16(numD),
			CodeSizes:          sX[:],
			LiteralLengthSizes: sLL,
			DistanceSizes:      sD,
		},
	})

	fr.hLL = &fr.hd1
	fr.hD = &fr.hd2
	return true
}

func (fr *Reader) decodeX(sym huffman.Symbol, combinedLengths []byte, i uint, total uint) (uint, bool) {
	switch {
	case sym < 16:
		// next output symbol has length of sym bits
		combinedLengths[i] = byte(sym)
		i++

	case sym == 16:
		// next 3 .. 6 output symbols have length equal to previous output symbol
		if i == 0 {
			fr.corruptf("attempt to repeat -1'st length")
			return i, false
		}

		if ok := fr.inputBitsFill(2); !ok {
			fr.propagateError(true)
			return i, false
		}

		out := fr.inputBitsRead(2)
		count := 3 + uint(out)
		if count > (total - i) {
			fr.corruptf("attempt to repeat %d times but only %d codes remain", count, total-i)
			return i, false
		}

		lastLength := combinedLengths[i-1]
		for count != 0 {
			combinedLengths[i] = lastLength
			i++
			count--
		}

	case sym == 17:
		// next 3 .. 10 output symbols have length of 0 bits
		if ok := fr.inputBitsFill(3); !ok {
			fr.propagateError(true)
			return i, false
		}

		out := fr.inputBitsRead(3)
		count := 3 + uint(out)
		if count > (total - i) {
			fr.corruptf("attempt to repeat %d times but only %d codes remain", count, total-i)
			return i, false
		}

		i += count

	case sym == 18:
		// next 11 .. 138 output symbols have length of 0 bits
		if ok := fr.inputBitsFill(7); !ok {
			fr.propagateError(true)
			return i, false
		}

		out := fr.inputBitsRead(7)
		count := 11 + uint(out)
		if count > (total - i) {
			fr.corruptf("attempt to repeat %d times but only %d codes remain", count, total-i)
			return i, false
		}

		i += count
	}
	return i, true
}

func (fr *Reader) readSymbol(hdec *huffman.Decoder) (symbol huffman.Symbol, ok bool) {
	min := hdec.MinSize()
	max := hdec.MaxSize()
	numBits := min
	for numBits <= max {
		if ok := fr.inputBitsFill(numBits); !ok {
			return huffman.InvalidSymbol, false
		}

		out := fr.inputBitsPeek(numBits)
		hc := huffman.MakeCode(numBits, uint32(out))

		symbol, newMin, newMax := hdec.Decode(hc)
		if symbol >= 0 {
			fr.inputBitsCommit(numBits)
			return symbol, true
		}
		if newMax == 0 {
			return symbol, false
		}
		numBits = newMin
	}
	return huffman.InvalidSymbol, false
}

func (fr *Reader) inputBufferFill() {
	if fr.inputError != nil {
		return
	}

	if fr.stopIsForced {
		fr.inputError = io.EOF
		return
	}

	_, err := fr.inputBuffer.ReadFrom(fr.r)
	fr.inputError = err
}

func (fr *Reader) inputBufferRead(p []byte) ([]byte, bool) {
	fr.inputBitsDiscard()

	pLen := uint(len(p))
	if pLen == 0 {
		return p, true
	}

	pIndex := uint(0)
	for pIndex < pLen {
		if fr.inputBuffer.IsEmpty() {
			fr.inputBufferFill()
			if fr.inputBuffer.IsEmpty() {
				break
			}
		}
		nn, _ := fr.inputBuffer.Read(p[pIndex:])
		pIndex += uint(nn)
	}

	fr.inputBytesTotal += uint64(pIndex)
	fr.inputBytesStream += uint64(pIndex)
	fr.inputBytesCRC32.Write(p[:pIndex])
	return p[:pIndex], (pIndex == pLen)
}

func (fr *Reader) inputBufferReadU16(bo binary.ByteOrder) (u16 uint16, ok bool) {
	p, pOK := fr.inputBufferRead(fr.tmp[:2])
	if pOK {
		u16 = bo.Uint16(p)
		ok = true
	}
	return
}

func (fr *Reader) inputBufferReadU32(bo binary.ByteOrder) (u32 uint32, ok bool) {
	p, pOK := fr.inputBufferRead(fr.tmp[:4])
	if pOK {
		u32 = bo.Uint32(p)
		ok = true
	}
	return
}

func (fr *Reader) inputBufferReadStringZ() (str string, ok bool) {
	bb := bufferpool.Get()
	defer bufferpool.Put(bb)

	for {
		if fr.inputBuffer.IsEmpty() {
			fr.inputBufferFill()
			if fr.inputBuffer.IsEmpty() {
				break
			}
		}

		ch, _ := fr.inputBuffer.ReadByte()
		fr.inputBytesTotal++
		fr.inputBytesStream++
		fr.inputBytesCRC32.Write([]byte{ch})

		if ch == 0 {
			ok = true
			break
		}

		bb.WriteByte(ch)
	}

	str = bb.String()
	return
}

func (fr *Reader) inputBitsFill(atLeast byte) bool {
	limit := byte(bitsPerBlock - bitsPerByte)
	assert.Assertf(atLeast <= limit, "atLeast %d > limit %d", atLeast, limit)

	for fr.ibLen < atLeast {
		if fr.inputBuffer.IsEmpty() {
			fr.inputBufferFill()
			if fr.inputBuffer.IsEmpty() {
				return false
			}
		}

		ch, _ := fr.inputBuffer.ReadByte()
		fr.inputBytesTotal++
		fr.inputBytesStream++
		fr.inputBytesCRC32.Write([]byte{ch})

		fr.ibBlock |= (block(ch) << fr.ibLen)
		fr.ibLen += bitsPerByte
	}

	return true
}

func (fr *Reader) inputBitsRead(wantLen byte) block {
	out := fr.inputBitsPeek(wantLen)
	fr.inputBitsCommit(wantLen)
	return out
}

func (fr *Reader) inputBitsPeek(wantLen byte) block {
	assert.Assertf(wantLen <= fr.ibLen, "wantLen %d > ibLen %d", wantLen, fr.ibLen)
	return (fr.ibBlock & makeMask(wantLen))
}

func (fr *Reader) inputBitsCommit(wantLen byte) {
	assert.Assertf(wantLen <= fr.ibLen, "wantLen %d > ibLen %d", wantLen, fr.ibLen)

	fr.ibBlock = (fr.ibBlock >> wantLen)
	fr.ibLen -= wantLen
}

func (fr *Reader) inputBitsDiscard() {
	fr.ibBlock = 0
	fr.ibLen = 0
}

func (fr *Reader) decodeBufferWrite(p []byte) bool {
	length := uint(len(p))
	for length > 0 {
		fr.decodeBufferTryFlush()
		nn, _ := fr.decodeBuffer.Write(p)
		_, _ = fr.window.Write(p[:nn])
		p = p[nn:]
		length -= uint(nn)
	}
	return true
}

func (fr *Reader) decodeBufferWriteByte(ch byte) bool {
	fr.decodeBufferTryFlush()
	err := fr.decodeBuffer.WriteByte(ch)
	if err == nil {
		_ = fr.window.WriteByte(ch)
	}
	return (err == nil)
}

func (fr *Reader) decodeBufferTryFlush() {
	if fr.decodeBuffer.IsFull() {
		fr.decodeBufferMustFlush()
	}
}

func (fr *Reader) decodeBufferMustFlush() {
	if fr.stopIsForced {
		fr.decodeBuffer.Clear()
		return
	}

	size := fr.decodeBuffer.Size()
	for !fr.decodeBuffer.IsEmpty() {
		fr.mu1.Unlock()
		fr.mu2.Lock()
		for fr.outputBuffer.IsFull() {
			fr.cv2.Wait()
		}
		fr.mu2.Unlock()
		fr.mu1.Lock()

		if fr.stopIsForced {
			fr.decodeBuffer.Clear()
			return
		}

		p := fr.decodeBuffer.PrepareBulkRead(size)

		fr.mu2.Lock()
		nn, _ := fr.outputBuffer.Write(p)
		if nn != 0 {
			fr.cv1.Signal()
		}
		fr.mu2.Unlock()

		if nn != 0 {
			fr.outputBytesTotal += uint64(nn)
			fr.outputBytesStream += uint64(nn)
			_, _ = fr.outputBytesAdler32.Write(p[:nn])
			_, _ = fr.outputBytesCRC32.Write(p[:nn])

			fr.decodeBuffer.CommitBulkRead(uint(nn))
		}
	}
}

func (fr *Reader) propagateError(eofIsError bool) {
	if fr.inputError == nil {
		return
	}

	err := fr.inputError
	if eofIsError && err == io.EOF {
		err = io.ErrUnexpectedEOF
	}
	fr.closeWithError(err)
}

func (fr *Reader) corruptf(format string, v ...interface{}) {
	message := fmt.Sprintf(format, v...)
	err := CorruptInputError{
		OffsetTotal:  fr.inputBytesTotal,
		OffsetStream: fr.inputBytesStream,
		Problem:      message,
	}
	fr.closeWithError(err)
}

func (fr *Reader) closeWithError(err error) {
	if fr.outputErrorWasSet {
		return
	}

	fr.mu2.Lock()
	fr.outputError = err
	fr.cv1.Broadcast()
	fr.mu2.Unlock()

	fr.outputErrorWasSet = true
}

func (fr *Reader) sendEvent(event Event) {
	event.InputBytesTotal = fr.inputBytesTotal
	event.InputBytesStream = fr.inputBytesStream
	event.OutputBytesTotal = fr.outputBytesTotal
	event.OutputBytesStream = fr.outputBytesStream
	event.NumStreams = fr.numStreams
	event.Format = fr.actualFormat
	for _, tr := range fr.tracers {
		tr.OnEvent(event)
	}
}
