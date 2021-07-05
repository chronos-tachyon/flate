package flate

import (
	"bytes"
	"encoding/hex"
	"io"
	"testing"
)

func TestReader(t *testing.T) {
	type testRow struct {
		name         string
		format       Format
		dict         []byte
		compressed   []byte
		decompressed []byte
	}

	var testData = [...]testRow{
		{
			name:         "lipsum",
			format:       RawFormat,
			dict:         nil,
			compressed:   mustDecodeHex("04c0d10904210c04d056a680c32aee739b90382c036a2489fdef7b3cb8a0937761f8f440aad017eb07f39db462dd401f3a4ad37ec1a96af8fba6e1ce0a19b37d010000ffff"),
			decompressed: []byte("Lorem ipsum dolor sit amet, consectetur adipiscing elit. Donec ultrices."),
		},
		{
			name:         "pangram",
			format:       RawFormat,
			dict:         nil,
			compressed:   mustDecodeHex("0a2ec8c8ccab50c84f5348ca494cce56282c4d2c2aa9d251c82a4d494f55c8ad5428cb2fd703040000ffff"),
			decompressed: []byte("Sphinx of black quartz, judge my vow."),
		},
		{
			name:         "repetitive-1",
			format:       RawFormat,
			dict:         nil,
			compressed:   mustDecodeHex("52484c4a4e51484d4bcf406221b8083140000000ffff"),
			decompressed: []byte(" abcd efgh abcd efgh efgh abcd abcd efgh "),
		},
		{
			name:         "repetitive-2",
			format:       RawFormat,
			dict:         []byte(" abcd efgh "),
			compressed:   mustDecodeHex("42622258082e420c100000ffff"),
			decompressed: []byte(" abcd efgh abcd efgh efgh abcd abcd efgh "),
		},
		{
			name:         "big-lipsum-zraw",
			format:       RawFormat,
			dict:         nil,
			compressed:   mustReadFile(testdataFS, "testdata/lipsum.txt.zraw"),
			decompressed: mustReadFile(testdataFS, "testdata/lipsum.txt"),
		},
		{
			name:         "big-lipsum-zlib",
			format:       ZlibFormat,
			dict:         nil,
			compressed:   mustReadFile(testdataFS, "testdata/lipsum.txt.zlib"),
			decompressed: mustReadFile(testdataFS, "testdata/lipsum.txt"),
		},
		{
			name:         "big-lipsum-gzip",
			format:       GZIPFormat,
			dict:         nil,
			compressed:   mustReadFile(testdataFS, "testdata/lipsum.txt.gz"),
			decompressed: mustReadFile(testdataFS, "testdata/lipsum.txt"),
		},
		{
			name:         "big-lipsum-zraw-auto",
			format:       DefaultFormat,
			dict:         nil,
			compressed:   mustReadFile(testdataFS, "testdata/lipsum.txt.zraw"),
			decompressed: mustReadFile(testdataFS, "testdata/lipsum.txt"),
		},
		{
			name:         "big-lipsum-zlib-auto",
			format:       DefaultFormat,
			dict:         nil,
			compressed:   mustReadFile(testdataFS, "testdata/lipsum.txt.zlib"),
			decompressed: mustReadFile(testdataFS, "testdata/lipsum.txt"),
		},
		{
			name:         "big-lipsum-gzip-auto",
			format:       DefaultFormat,
			dict:         nil,
			compressed:   mustReadFile(testdataFS, "testdata/lipsum.txt.gz"),
			decompressed: mustReadFile(testdataFS, "testdata/lipsum.txt"),
		},
	}

	r := NewReader(eofReader{})

	for _, vector := range testData {
		t.Run(vector.name, func(t *testing.T) {
			src := bytes.NewReader(vector.compressed)
			r.Reset(
				src,
				WithFormat(vector.format),
				WithDictionary(vector.dict),
			)

			output, err := io.ReadAll(r)
			if err != nil {
				t.Errorf("Read failed: %v", err)
				return
			}

			actual := output
			expect := vector.decompressed
			if !bytes.Equal(actual, expect) {
				actualLen := uint(len(actual))
				expectLen := uint(len(expect))

				minLen := actualLen
				maxLen := actualLen
				if minLen > expectLen {
					minLen = expectLen
				}
				if maxLen < expectLen {
					maxLen = expectLen
				}

				hasFirst := false
				var first uint
				var count uint
				for index := uint(0); index < minLen; index++ {
					if actual[index] == expect[index] {
						continue
					}
					if !hasFirst {
						hasFirst = true
						first = index
					}
					count++
				}

				if !hasFirst {
					first = minLen
				}
				count += (maxLen - minLen)

				t.Errorf("unexpected diff: first change at offset %d, %d bytes changed, lengths %d vs %d", first, count, actualLen, expectLen)
				if maxLen < 32 {
					t.Logf("expect: %s", hex.EncodeToString(expect))
					t.Logf("actual: %s", hex.EncodeToString(actual))
				}
			}
		})
	}
}

func BenchmarkReader(b *testing.B) {
	raw := mustReadFile(testdataFS, "testdata/lipsum.txt.zraw")
	txt := mustReadFile(testdataFS, "testdata/lipsum.txt")
	r := NewReader(eofReader{}, WithFormat(RawFormat))
	for n := 0; n < b.N; n++ {
		src := bytes.NewReader(raw)
		r.Reset(src)
		nn, err := io.Copy(io.Discard, r)
		if err != nil {
			b.Fatalf("io.Copy failed: %v", err)
		}
		if nn != int64(len(txt)) {
			b.Errorf("wrong length: expected %d, got %d", len(txt), nn)
		}
	}
}
