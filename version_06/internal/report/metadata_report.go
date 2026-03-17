package report

import (
	"fmt"
	"os"

	"fwparse/internal/firmware"
)

func SaveMetadataReport(outPath string, meta *firmware.Metadata) error {
	file, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer file.Close()

	fmt.Fprintln(file, "Firmware Metadata Values")
	fmt.Fprintln(file, "---")
	fmt.Fprintf(file, "CW_FAMILY = %s\n", meta.Family)
	fmt.Fprintf(file, "CW_IMAGE = %s\n", meta.Image)
	fmt.Fprintf(file, "CW_VERSION = %s\n", meta.Version)
	fmt.Fprintf(file, "CW_FULL_VERSION = %s\n", meta.FullVersion)
	fmt.Fprintf(file, "CW_DESCRIPTION = %s\n", meta.Description)

	return nil
}
