package report

import (
	"fmt"
	"os"

	"fwparse/internal/model"
)

func SaveFilesystemLocationReport(
	outPath string,
	firmwarePath string,
	firmwareSize int64,
	fsOffset int64,
	sb *model.SquashfsSuperblock,
) error {
	file, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer file.Close()

	fmt.Fprintln(file, "Filesystem Location Report")
	fmt.Fprintln(file, "---")
	fmt.Fprintf(file, "Firmware image: %s\n", firmwarePath)
	fmt.Fprintf(file, "Firmware size: %d bytes\n", firmwareSize)
	fmt.Fprintf(file, "Filesystem offset: 0x%X\n", fsOffset)
	fmt.Fprintf(file, "Filesystem size (bytes_used): %d\n", sb.BytesUsed)
	fmt.Fprintf(file, "SquashFS version: %d.%d\n", sb.VersionMajor, sb.VersionMinor)
	fmt.Fprintf(file, "Block size: %d\n", sb.BlockSize)

	return nil
}
