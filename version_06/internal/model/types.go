package model

type FirmwareHeader struct {
	Data []byte
}

type Section struct {
	Name   string
	Offset int
	Size   int
}

type Node struct {
	Name     string
	Offset   int64
	Size     int
	Type     string
	Details  string
	Children []*Node
}

type PkgField struct {
	Offset int
	Tag    byte
	Length int
	Value  []byte
}

type SquashfsSuperblock struct {
	Magic               uint32
	InodeCount          uint32
	ModTime             uint32
	BlockSize           uint32
	FragmentCount       uint32
	Compression         uint16
	BlockLog            uint16
	Flags               uint16
	IdCount             uint16
	VersionMajor        uint16
	VersionMinor        uint16
	RootInode           uint64
	BytesUsed           uint64
	IdTableStart        uint64
	XattrIdTableStart   uint64
	InodeTableStart     uint64
	DirectoryTableStart uint64
	FragmentTableStart  uint64
	LookupTableStart    uint64
}
