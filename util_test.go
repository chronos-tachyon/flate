package flate

import (
	"encoding/hex"
	"fmt"
	"io/fs"
	"strings"
)

func mustDecodeHex(str string) []byte {
	raw, err := hex.DecodeString(str)
	if err != nil {
		panic(err)
	}
	return raw
}

func mustReadFile(filesystem fs.FS, path string) []byte {
	raw, err := fs.ReadFile(filesystem, path)
	if err != nil {
		panic(err)
	}
	return raw
}

func hexDump(p []byte) []string {
	length := uint(len(p))
	lines := make([]string, 0, (length+15)>>4)
	var offset uint
	var buf strings.Builder
	for (offset + 16) <= length {
		buf.Reset()
		fmt.Fprintf(&buf, "%08x|", offset)
		for i := uint(0); i < 16; i++ {
			index := offset + i
			ch := p[index]
			fmt.Fprintf(&buf, " %02x", ch)
			if i == 7 {
				buf.WriteByte(' ')
			}
		}
		lines = append(lines, buf.String())
		offset += 16
	}
	if offset < length || offset == 0 {
		buf.Reset()
		fmt.Fprintf(&buf, "%08x|", offset)
		for i := uint(0); i < 16; i++ {
			index := offset + i
			if index < length {
				ch := p[index]
				fmt.Fprintf(&buf, " %02x", ch)
			} else {
				buf.WriteString(" --")
			}
			if i == 7 {
				buf.WriteByte(' ')
			}
		}
		lines = append(lines, buf.String())
	}
	return lines
}

func hexDiff(a, b []byte) []string {
	aLines := hexDump(a)
	bLines := hexDump(b)

	aLen := uint(len(aLines))
	bLen := uint(len(bLines))
	minLen := aLen
	if minLen > bLen {
		minLen = bLen
	}

	diffLines := make([]string, 0, aLen+bLen)
	for i := uint(0); i < minLen; i++ {
		aLine := aLines[i]
		bLine := bLines[i]
		if aLine == bLine {
			continue
		}
		diffLines = append(diffLines, "-"+aLine)
		diffLines = append(diffLines, "+"+bLine)
	}
	for i := minLen; i < aLen; i++ {
		aLine := aLines[i]
		diffLines = append(diffLines, "-"+aLine)
	}
	for i := minLen; i < bLen; i++ {
		bLine := bLines[i]
		diffLines = append(diffLines, "+"+bLine)
	}
	return diffLines
}

func tabify(lines []string) string {
	var buf strings.Builder
	for _, line := range lines {
		buf.WriteByte('\n')
		buf.WriteByte('\t')
		buf.WriteString(line)
	}
	return buf.String()
}
