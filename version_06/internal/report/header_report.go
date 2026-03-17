package report

import (
	"fmt"
	"os"

	"fwparse/internal/model"
)

func SaveHeaderReport(
	outPath string,
	firmwareSize int64,
	headerSize int,
	sections []model.Section,
	ranges []string,
	boundaryErrors []string,
) error {
	file, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer file.Close()

	fmt.Fprintln(file, "Firmware Header Section Map")
	fmt.Fprintln(file, "---")
	fmt.Fprintln(file)

	fmt.Fprintf(file, "Firmware size: %d bytes\n", firmwareSize)
	fmt.Fprintf(file, "Header size: %d bytes\n", headerSize)
	fmt.Fprintln(file)

	fmt.Fprintln(file, "Found sections:")
	for _, s := range sections {
		fmt.Fprintf(file, "Section: %s | Offset: 0x%X | Size: %d bytes\n", s.Name, s.Offset, s.Size)
	}

	fmt.Fprintln(file)
	fmt.Fprintln(file, "Approximate section ranges:")
	for _, r := range ranges {
		fmt.Fprintln(file, r)
	}

	fmt.Fprintln(file)
	fmt.Fprintln(file, "Boundary checks:")
	if len(boundaryErrors) == 0 {
		fmt.Fprintln(file, "All section boundary checks passed")
	} else {
		for _, e := range boundaryErrors {
			fmt.Fprintln(file, e)
		}
	}

	return nil
}
