package debug

import (
	"fmt"
	"io"
	"os"

	"fwparse/internal/model"
)

func previewValue(data []byte) string {
	var out []byte

	for _, b := range data {
		if b >= 32 && b <= 126 {
			out = append(out, b)
		}
	}

	if len(out) == 0 {
		return "(binary)"
	}

	if len(out) > 60 {
		return string(out[:60]) + "..."
	}

	return string(out)
}

func SavePkgParsedFields(fields []model.PkgField, baseOffset int64) {
	file, err := os.Create("output/pkg_parsed_fields.txt")
	if err != nil {
		fmt.Println("Error creating pkg parsed fields file:", err)
		return
	}
	defer file.Close()

	fmt.Fprintln(file, "PKG Parsed Fields")
	fmt.Fprintln(file, "---")
	fmt.Fprintf(file, "Base offset: 0x%X\n\n", baseOffset)

	for i, f := range fields {
		fmt.Fprintf(
			file,
			"Field %d | field_offset=0x%X | tag=0x%02X | length=%d | preview=%s\n",
			i+1,
			baseOffset+int64(f.Offset),
			f.Tag,
			f.Length,
			previewValue(f.Value),
		)
	}
}

func SavePkgOffsetAnalysis(reader io.ReaderAt, fileSize int64, offset int64) {
	readSize := 8192

	remaining := fileSize - offset
	if remaining <= 0 {
		fmt.Println("No bytes left for pkg offset analysis")
		return
	}

	if int64(readSize) > remaining {
		readSize = int(remaining)
	}

	data := make([]byte, readSize)
	n, err := reader.ReadAt(data, offset)
	if err != nil && err != io.EOF {
		fmt.Println("Error reading pkg offset analysis:", err)
		return
	}

	data = data[:n]

	hexFile, err := os.Create("output/pkg_header_hex_offsets.txt")
	if err != nil {
		fmt.Println("Error creating pkg header hex offsets file:", err)
		return
	}
	defer hexFile.Close()

	fmt.Fprintln(hexFile, "PKG Header Hex With Offsets")
	fmt.Fprintln(hexFile, "---")
	fmt.Fprintf(hexFile, "Base offset: 0x%X\n\n", offset)

	limit := 256
	if len(data) < limit {
		limit = len(data)
	}

	for i := 0; i < limit; i += 16 {
		lineEnd := i + 16
		if lineEnd > limit {
			lineEnd = limit
		}

		fmt.Fprintf(hexFile, "0x%04X: ", i)
		for j := i; j < lineEnd; j++ {
			fmt.Fprintf(hexFile, "%02X ", data[j])
		}
		fmt.Fprintln(hexFile)
	}

	fragmentsFile, err := os.Create("output/pkg_printable_fragments.txt")
	if err != nil {
		fmt.Println("Error creating pkg printable fragments file:", err)
		return
	}
	defer fragmentsFile.Close()

	fmt.Fprintln(fragmentsFile, "PKG Printable Fragments")
	fmt.Fprintln(fragmentsFile, "---")
	fmt.Fprintf(fragmentsFile, "Base offset: 0x%X\n\n", offset)

	var current []byte
	startOffset := 0

	for i, b := range data {
		if b >= 32 && b <= 126 {
			if len(current) == 0 {
				startOffset = i
			}
			current = append(current, b)
		} else {
			if len(current) >= 4 {
				fmt.Fprintf(fragmentsFile, "0x%04X -> %s\n", startOffset, string(current))
			}
			current = nil
		}
	}

	if len(current) >= 4 {
		fmt.Fprintf(fragmentsFile, "0x%04X -> %s\n", startOffset, string(current))
	}
}
