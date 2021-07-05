package flate

import (
	"fmt"

	"github.com/chronos-tachyon/enumhelper"
)

// OSType indicates the OS or filesystem type on which a gzip-compressed file
// originated.
type OSType byte

const (
	// OSTypeUnknown indicates that the originating OS and/or filesystem is
	// not known.
	OSTypeUnknown OSType = iota

	// OSTypeFAT indicates that the file originated on an MS-DOS or Microsoft
	// Windows host with a FAT-based filesystem such as FAT16 or FAT32.
	OSTypeFAT

	// OSTypeAmiga indicates that the file originated on an Amiga host.
	OSTypeAmiga

	// OSTypeVMS indicates that the file originated on a VMS host.
	OSTypeVMS

	// OSTypeUnix indicates that the file originated on a Unix host, or on a
	// host with a Unix-like filesystem.
	OSTypeUnix

	// OSTypeVMCMS indicates that the file originated on a VM/CMS host.
	OSTypeVMCMS

	// OSTypeAtariTOS indicates that the file originated on an Atari TOS
	// host.
	OSTypeAtariTOS

	// OSTypeHPFS indicates that the file originated on an OS/2 host with an
	// HPFS filesystem.
	OSTypeHPFS

	// OSTypeMacintosh indicates that the file originated on an Apple
	// Macintosh host (Classic or Mac OS X) with an HFS or HFS+
	// filesystem.
	OSTypeMacintosh

	// OSTypeZSystem indicates that the file originated on an IBM Z/System host.
	OSTypeZSystem

	// OSTypeCPM indicates that the file originated on a CP/M host.
	OSTypeCPM

	// OSTypeTOPS20 indicates that the file originated on a TOPS-20 host.
	OSTypeTOPS20

	// OSTypeNTFS indicates that the file originated on a Microsoft Windows NT
	// host with an NTFS filesystem.
	OSTypeNTFS

	// OSTypeQDOS indicates that the file originated on a QDOS host.
	OSTypeQDOS

	// OSTypeAcornRISCOS indicates that the file originated on an Acorn RISCOS
	// host.
	OSTypeAcornRISCOS
)

var osTypeData = []enumhelper.EnumData{
	{
		GoName: "OSTypeUnknown",
		Name:   "unknown",
		JSON:   []byte(`"OSTypeUnknown"`),
	},
	{
		GoName: "OSTypeFAT",
		Name:   "FAT filesystem",
		JSON:   []byte(`"OSTypeFAT"`),
	},
	{
		GoName: "OSTypeAmiga",
		Name:   "Amiga",
		JSON:   []byte(`"OSTypeAmiga"`),
	},
	{
		GoName: "OSTypeVMS",
		Name:   "VMS",
		JSON:   []byte(`"OSTypeVMS"`),
	},
	{
		GoName: "OSTypeUnix",
		Name:   "Unix",
		JSON:   []byte(`"OSTypeUnix"`),
	},
	{
		GoName: "OSTypeVMCMS",
		Name:   "VM/CMS",
		JSON:   []byte(`"OSTypeVMCMS"`),
	},
	{
		GoName: "OSTypeAtariTOS",
		Name:   "Atari TOS",
		JSON:   []byte(`"OSTypeAtariTOS"`),
	},
	{
		GoName: "OSTypeHPFS",
		Name:   "HPFS filesystem",
		JSON:   []byte(`"OSTypeHPFS"`),
	},
	{
		GoName: "OSTypeMacintosh",
		Name:   "Macintosh",
		JSON:   []byte(`"OSTypeMacintosh"`),
	},
	{
		GoName: "OSTypeZSystem",
		Name:   "Z-System",
		JSON:   []byte(`"OSTypeZSystem"`),
	},
	{
		GoName: "OSTypeCPM",
		Name:   "CP/M",
		JSON:   []byte(`"OSTypeCPM"`),
	},
	{
		GoName: "OSTypeTOPS20",
		Name:   "TOPS-20",
		JSON:   []byte(`"OSTypeTOPS20"`),
	},
	{
		GoName: "OSTypeNTFS",
		Name:   "NTFS filesystem",
		JSON:   []byte(`"OSTypeNTFS"`),
	},
	{
		GoName: "OSTypeQDOS",
		Name:   "QDOS",
		JSON:   []byte(`"OSTypeQDOS"`),
	},
	{
		GoName: "OSTypeAcornRISCOS",
		Name:   "Acorn RISCOS",
		JSON:   []byte(`"OSTypeAcornRISCOS"`),
	},
}

var gzipOSTypeDecodeTable = map[byte]OSType{
	0x00: OSTypeFAT,
	0x01: OSTypeAmiga,
	0x02: OSTypeVMS,
	0x03: OSTypeUnix,
	0x04: OSTypeVMCMS,
	0x05: OSTypeAtariTOS,
	0x06: OSTypeHPFS,
	0x07: OSTypeMacintosh,
	0x08: OSTypeZSystem,
	0x09: OSTypeCPM,
	0x0a: OSTypeTOPS20,
	0x0b: OSTypeNTFS,
	0x0c: OSTypeQDOS,
	0x0d: OSTypeAcornRISCOS,
}

var gzipOSTypeEncodeTable = map[OSType]byte{
	OSTypeFAT:         0x00,
	OSTypeAmiga:       0x01,
	OSTypeVMS:         0x02,
	OSTypeUnix:        0x03,
	OSTypeVMCMS:       0x04,
	OSTypeAtariTOS:    0x05,
	OSTypeHPFS:        0x06,
	OSTypeMacintosh:   0x07,
	OSTypeZSystem:     0x08,
	OSTypeCPM:         0x09,
	OSTypeTOPS20:      0x0a,
	OSTypeNTFS:        0x0b,
	OSTypeQDOS:        0x0c,
	OSTypeAcornRISCOS: 0x0d,
}

// IsValid returns true if o is a valid OSType constant.
func (o OSType) IsValid() bool {
	return o >= OSTypeUnknown && o <= OSTypeAcornRISCOS
}

// GoString returns the Go string representation of this OSType constant.
func (o OSType) GoString() string {
	return enumhelper.DereferenceEnumData("OSType", osTypeData, uint(o)).GoName
}

// String returns the string representation of this OSType constant.
func (o OSType) String() string {
	return enumhelper.DereferenceEnumData("OSType", osTypeData, uint(o)).Name
}

// MarshalJSON returns the JSON representation of this OSType constant.
func (o OSType) MarshalJSON() ([]byte, error) {
	return enumhelper.MarshalEnumToJSON("OSType", osTypeData, uint(o))
}

var _ fmt.GoStringer = OSType(0)
var _ fmt.Stringer = OSType(0)
