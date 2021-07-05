package flate

import (
	"encoding/binary"
	"time"
)

// Header is a collection of fields which may be present in the headers of a
// gzip or zlib stream.
type Header struct {
	FileName      string
	Comment       string
	LastModified  time.Time
	DataType      DataType
	OSType        OSType
	ExtraData     ExtraData
	WindowBits    WindowBits
	CompressLevel CompressLevel
}

// ExtraData represents a collection of records in a gzip ExtraData header.
type ExtraData struct {
	Records []ExtraDataRecord
}

// ExtraDataRecord represents a single record in a gzip ExtraData header.
type ExtraDataRecord struct {
	ID    [2]byte
	Bytes []byte
}

// Parse parses the given bytes as an ExtraData field.
func (xd *ExtraData) Parse(raw []byte) {
	*xd = ExtraData{}

	index := uint(0)
	length := uint(len(raw))
	for (index + 4) <= length {
		var rec ExtraDataRecord
		rec.ID[0] = raw[index+0]
		rec.ID[1] = raw[index+1]
		recLen := uint(binary.LittleEndian.Uint16(raw[index+2 : index+4]))
		index += 4
		rec.Bytes = raw[index : index+recLen]
		index += recLen
		xd.Records = append(xd.Records, rec)
	}
}

// AsBytes returns the binary representation of this ExtraData field.
func (xd *ExtraData) AsBytes() []byte {
	var length uint
	for _, rec := range xd.Records {
		recLen := uint(len(rec.Bytes))
		length += 4 + recLen
	}

	out := make([]byte, 0, length)
	for _, rec := range xd.Records {
		var tmp [2]byte
		binary.LittleEndian.PutUint16(tmp[:], uint16(len(rec.Bytes)))
		out = append(out, rec.ID[0], rec.ID[1], tmp[0], tmp[1])
		out = append(out, rec.Bytes...)
	}
	return out
}
