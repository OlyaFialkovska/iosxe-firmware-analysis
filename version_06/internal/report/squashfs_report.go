package report

import (
	"fmt"
	"os"

	"fwparse/internal/model"
)

func SaveSquashfsSuperblockReport(outPath string, offset int64, sb *model.SquashfsSuperblock) error {
	err := os.MkdirAll("../../output/squashfs_info", 0755)
	if err != nil {
		return err
	}

	file, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer file.Close()

	fmt.Fprintln(file, "SquashFS Superblock")
	fmt.Fprintln(file, "---")
	fmt.Fprintf(file, "Offset: 0x%X\n", offset)
	fmt.Fprintf(file, "Version: %d.%d\n", sb.VersionMajor, sb.VersionMinor)
	fmt.Fprintf(file, "Compression: %d\n", sb.Compression)
	fmt.Fprintf(file, "Block size: %d\n", sb.BlockSize)
	fmt.Fprintf(file, "Block log: %d\n", sb.BlockLog)
	fmt.Fprintf(file, "Inodes: %d\n", sb.InodeCount)
	fmt.Fprintf(file, "Fragments: %d\n", sb.FragmentCount)
	fmt.Fprintf(file, "Bytes used: %d\n", sb.BytesUsed)
	fmt.Fprintf(file, "Root inode: 0x%X\n", sb.RootInode)
	fmt.Fprintf(file, "Inode table start: 0x%X\n", sb.InodeTableStart)
	fmt.Fprintf(file, "Directory table start: 0x%X\n", sb.DirectoryTableStart)
	fmt.Fprintf(file, "Fragment table start: 0x%X\n", sb.FragmentTableStart)
	fmt.Fprintf(file, "Lookup table start: 0x%X\n", sb.LookupTableStart)

	return nil
}
