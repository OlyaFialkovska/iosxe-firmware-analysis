package squashfs

import (
	"bytes"
	"fmt"
	"io"

	"fwparse/internal/model"
)

func ReadSquashfsSize(reader io.ReaderAt, offset int64) int64 {
	header := make([]byte, 96)

	_, err := reader.ReadAt(header, offset)
	if err != nil {
		fmt.Println("Error reading squashfs superblock:", err)
		return 0
	}

	if len(header) < 48 {
		return 0
	}

	if !bytes.Equal(header[0:4], []byte("hsqs")) {
		return 0
	}

	var size int64
	for i := 0; i < 8; i++ {
		size |= int64(header[40+i]) << (8 * i)
	}

	return size
}

func ReadSquashfsSuperblock(reader io.ReaderAt, offset int64) (*model.SquashfsSuperblock, error) {
	buf := make([]byte, 96)

	_, err := reader.ReadAt(buf, offset)
	if err != nil {
		return nil, err
	}

	if !bytes.Equal(buf[0:4], []byte("hsqs")) {
		return nil, fmt.Errorf("invalid squashfs magic")
	}

	sb := &model.SquashfsSuperblock{}

	sb.Magic = uint32(buf[0]) | uint32(buf[1])<<8 | uint32(buf[2])<<16 | uint32(buf[3])<<24
	sb.InodeCount = uint32(buf[4]) | uint32(buf[5])<<8 | uint32(buf[6])<<16 | uint32(buf[7])<<24
	sb.ModTime = uint32(buf[8]) | uint32(buf[9])<<8 | uint32(buf[10])<<16 | uint32(buf[11])<<24
	sb.BlockSize = uint32(buf[12]) | uint32(buf[13])<<8 | uint32(buf[14])<<16 | uint32(buf[15])<<24
	sb.FragmentCount = uint32(buf[16]) | uint32(buf[17])<<8 | uint32(buf[18])<<16 | uint32(buf[19])<<24

	sb.Compression = uint16(buf[20]) | uint16(buf[21])<<8
	sb.BlockLog = uint16(buf[22]) | uint16(buf[23])<<8
	sb.Flags = uint16(buf[24]) | uint16(buf[25])<<8
	sb.IdCount = uint16(buf[26]) | uint16(buf[27])<<8
	sb.VersionMajor = uint16(buf[28]) | uint16(buf[29])<<8
	sb.VersionMinor = uint16(buf[30]) | uint16(buf[31])<<8

	read64 := func(start int) uint64 {
		var value uint64
		for i := 0; i < 8; i++ {
			value |= uint64(buf[start+i]) << (8 * i)
		}
		return value
	}

	sb.RootInode = read64(32)
	sb.BytesUsed = read64(40)
	sb.IdTableStart = read64(48)
	sb.XattrIdTableStart = read64(56)
	sb.InodeTableStart = read64(64)
	sb.DirectoryTableStart = read64(72)
	sb.FragmentTableStart = read64(80)
	sb.LookupTableStart = read64(88)

	return sb, nil
}
