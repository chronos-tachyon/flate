package flate

import (
	"bytes"
	"io"
	"io/ioutil"
	"testing"
)

func TestWriter(t *testing.T) {
	type testRow struct {
		name     string
		format   Format
		strategy Strategy
		clevel   CompressLevel
		mlevel   MemoryLevel
		wbits    WindowBits
		dict     []byte
		input    []byte
	}

	smallLipsum := []byte("Lorem ipsum dolor sit amet, consectetur adipiscing elit. Donec ultrices.")
	pangram := []byte("Sphinx of black quartz, judge my vow.")
	repetitive := []byte(" abcd efgh abcd efgh efgh abcd abcd efgh ")
	repetitiveDict := []byte(" abcd efgh ")
	bigLipsum := mustReadFile(testdataFS, "testdata/lipsum.txt")
	fourMegZero := make([]byte, 4<<20)

	var testData = [...]testRow{
		{
			name:     "lipsum-zraw",
			format:   RawFormat,
			strategy: DefaultStrategy,
			clevel:   BestCompression,
			mlevel:   FastestMemory,
			wbits:    MaxWindowBits,
			dict:     nil,
			input:    smallLipsum,
		},
		{
			name:     "lipsum-zlib",
			format:   ZlibFormat,
			strategy: DefaultStrategy,
			clevel:   BestCompression,
			mlevel:   FastestMemory,
			wbits:    MaxWindowBits,
			dict:     nil,
			input:    smallLipsum,
		},
		{
			name:     "lipsum-gzip",
			format:   GZIPFormat,
			strategy: DefaultStrategy,
			clevel:   BestCompression,
			mlevel:   FastestMemory,
			wbits:    MaxWindowBits,
			dict:     nil,
			input:    smallLipsum,
		},
		{
			name:     "pangram-zraw",
			format:   RawFormat,
			strategy: DefaultStrategy,
			clevel:   BestCompression,
			mlevel:   FastestMemory,
			wbits:    MaxWindowBits,
			dict:     nil,
			input:    pangram,
		},
		{
			name:     "pangram-zlib",
			format:   ZlibFormat,
			strategy: DefaultStrategy,
			clevel:   BestCompression,
			mlevel:   FastestMemory,
			wbits:    MaxWindowBits,
			dict:     nil,
			input:    pangram,
		},
		{
			name:     "pangram-gzip",
			format:   GZIPFormat,
			strategy: DefaultStrategy,
			clevel:   BestCompression,
			mlevel:   FastestMemory,
			wbits:    MaxWindowBits,
			dict:     nil,
			input:    pangram,
		},
		{
			name:     "repetitive-1-zraw",
			format:   RawFormat,
			strategy: DefaultStrategy,
			clevel:   BestCompression,
			mlevel:   FastestMemory,
			wbits:    MaxWindowBits,
			dict:     nil,
			input:    repetitive,
		},
		{
			name:     "repetitive-1-zlib",
			format:   ZlibFormat,
			strategy: DefaultStrategy,
			clevel:   BestCompression,
			mlevel:   FastestMemory,
			wbits:    MaxWindowBits,
			dict:     nil,
			input:    repetitive,
		},
		{
			name:     "repetitive-1-gzip",
			format:   GZIPFormat,
			strategy: DefaultStrategy,
			clevel:   BestCompression,
			mlevel:   FastestMemory,
			wbits:    MaxWindowBits,
			dict:     nil,
			input:    repetitive,
		},
		{
			name:     "repetitive-2-zraw",
			format:   RawFormat,
			strategy: DefaultStrategy,
			clevel:   BestCompression,
			mlevel:   FastestMemory,
			wbits:    MaxWindowBits,
			dict:     repetitiveDict,
			input:    repetitive,
		},
		{
			name:     "repetitive-2-zlib",
			format:   ZlibFormat,
			strategy: DefaultStrategy,
			clevel:   BestCompression,
			mlevel:   FastestMemory,
			wbits:    MaxWindowBits,
			dict:     repetitiveDict,
			input:    repetitive,
		},
		{
			name:     "repetitive-2-gzip",
			format:   GZIPFormat,
			strategy: DefaultStrategy,
			clevel:   BestCompression,
			mlevel:   FastestMemory,
			wbits:    MaxWindowBits,
			dict:     repetitiveDict,
			input:    repetitive,
		},
		{
			name:     "big-lipsum-zraw",
			format:   RawFormat,
			strategy: DefaultStrategy,
			clevel:   BestCompression,
			mlevel:   FastestMemory,
			wbits:    MaxWindowBits,
			dict:     nil,
			input:    bigLipsum,
		},
		{
			name:     "big-lipsum-zlib",
			format:   ZlibFormat,
			strategy: DefaultStrategy,
			clevel:   BestCompression,
			mlevel:   FastestMemory,
			wbits:    MaxWindowBits,
			dict:     nil,
			input:    bigLipsum,
		},
		{
			name:     "big-lipsum-gzip",
			format:   GZIPFormat,
			strategy: DefaultStrategy,
			clevel:   BestCompression,
			mlevel:   FastestMemory,
			wbits:    MaxWindowBits,
			dict:     nil,
			input:    bigLipsum,
		},
		{
			name:     "big-lipsum-huff-m9-w15",
			format:   ZlibFormat,
			strategy: HuffmanOnlyStrategy,
			clevel:   DefaultCompression,
			mlevel:   FastestMemory,
			wbits:    MaxWindowBits,
			dict:     nil,
			input:    bigLipsum,
		},
		{
			name:     "big-lipsum-huffrle-m9-w15",
			format:   ZlibFormat,
			strategy: HuffmanRLEStrategy,
			clevel:   DefaultCompression,
			mlevel:   FastestMemory,
			wbits:    MaxWindowBits,
			dict:     nil,
			input:    bigLipsum,
		},
		{
			name:     "big-lipsum-c0-m9-w15",
			format:   ZlibFormat,
			strategy: DefaultStrategy,
			clevel:   NoCompression,
			mlevel:   FastestMemory,
			wbits:    MaxWindowBits,
			dict:     nil,
			input:    bigLipsum,
		},
		{
			name:     "big-lipsum-c1-m9-w15",
			format:   ZlibFormat,
			strategy: HuffmanRLEStrategy,
			clevel:   FastestCompression,
			mlevel:   FastestMemory,
			wbits:    MaxWindowBits,
			dict:     nil,
			input:    bigLipsum,
		},
		{
			name:     "big-lipsum-c9-m1-w15",
			format:   ZlibFormat,
			strategy: DefaultStrategy,
			clevel:   BestCompression,
			mlevel:   SmallestMemory,
			wbits:    MaxWindowBits,
			dict:     nil,
			input:    bigLipsum,
		},
		{
			name:     "big-lipsum-c9-m9-w8",
			format:   ZlibFormat,
			strategy: DefaultStrategy,
			clevel:   BestCompression,
			mlevel:   FastestMemory,
			wbits:    MinWindowBits,
			dict:     nil,
			input:    bigLipsum,
		},
		{
			name:     "big-lipsum-c9-m1-w8",
			format:   ZlibFormat,
			strategy: DefaultStrategy,
			clevel:   BestCompression,
			mlevel:   SmallestMemory,
			wbits:    MinWindowBits,
			dict:     nil,
			input:    bigLipsum,
		},
		{
			name:     "fourmegzero-huff-m9-w15",
			format:   GZIPFormat,
			strategy: HuffmanOnlyStrategy,
			clevel:   BestCompression,
			mlevel:   FastestMemory,
			wbits:    MaxWindowBits,
			dict:     nil,
			input:    fourMegZero,
		},
		{
			name:     "fourmegzero-huffrle-m9-w15",
			format:   GZIPFormat,
			strategy: HuffmanRLEStrategy,
			clevel:   BestCompression,
			mlevel:   FastestMemory,
			wbits:    MaxWindowBits,
			dict:     nil,
			input:    fourMegZero,
		},
		{
			name:     "fourmegzero-c0-m9-w15",
			format:   GZIPFormat,
			strategy: DefaultStrategy,
			clevel:   NoCompression,
			mlevel:   FastestMemory,
			wbits:    MaxWindowBits,
			dict:     nil,
			input:    fourMegZero,
		},
		{
			name:     "fourmegzero-c9-m9-w15",
			format:   GZIPFormat,
			strategy: DefaultStrategy,
			clevel:   BestCompression,
			mlevel:   FastestMemory,
			wbits:    MaxWindowBits,
			dict:     nil,
			input:    fourMegZero,
		},
	}

	fw := NewWriter(io.Discard)
	fr := NewReader(eofReader{})
	var buf bytes.Buffer

	for _, vector := range testData {
		t.Run(vector.name, func(t *testing.T) {
			buf.Reset()

			fw.Reset(
				&buf,
				WithFormat(vector.format),
				WithStrategy(vector.strategy),
				WithCompressLevel(vector.clevel),
				WithMemoryLevel(vector.mlevel),
				WithWindowBits(vector.wbits),
				WithDictionary(vector.dict),
			)

			originalSize := len(vector.input)

			nn, err := fw.Write(vector.input)
			if err != nil {
				t.Errorf("Write failed: %v", err)
				return
			}
			if nn != originalSize {
				t.Errorf("Write returned wrong length: expect %d, actual %d", originalSize, nn)
			}

			err = fw.Close()
			if err != nil {
				t.Errorf("Close failed: %v", err)
				return
			}

			fr.Reset(
				&buf,
				WithFormat(vector.format),
				WithDictionary(vector.dict),
			)

			raw, err := ioutil.ReadAll(fr)
			if err != nil {
				t.Errorf("Read failed: %v", err)
				return
			}

			decompressedSize := len(raw)

			if originalSize != decompressedSize {
				t.Errorf("Read returned wrong length: expect %d, actual %d", originalSize, decompressedSize)
			}

			if !bytes.Equal(raw, vector.input) {
				t.Error("Read returned wrong contents" + tabify(hexDiff(vector.input, raw)))
				t.Log("Compressed contents are" + tabify(hexDump(buf.Bytes())))
			}
		})
	}
}
