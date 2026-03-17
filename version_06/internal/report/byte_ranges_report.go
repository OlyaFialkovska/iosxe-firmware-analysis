package report

import (
	"fmt"
	"os"
)

func SaveByteRangesReport(
	outPath string,
	fileSize int64,
	headerSize int64,
	fsOffset int64,
	fsSize int64,
) error {
	file, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer file.Close()

	fmt.Fprintln(file, "Firmware Byte Ranges")
	fmt.Fprintln(file, "---")

	headerStart := int64(0)
	headerEnd := headerSize - 1

	fmt.Fprintf(
		file,
		"0x%08X - 0x%08X | %12d bytes | firmware header\n",
		headerStart,
		headerEnd,
		headerEnd-headerStart+1,
	)

	codeStart := headerSize
	codeEnd := fsOffset - 1
	if codeEnd >= codeStart {
		fmt.Fprintf(
			file,
			"0x%08X - 0x%08X | %12d bytes | boot/loader or non-filesystem region\n",
			codeStart,
			codeEnd,
			codeEnd-codeStart+1,
		)
	}

	fsStart := fsOffset
	fsEnd := fsOffset + fsSize - 1
	if fsSize > 0 {
		fmt.Fprintf(
			file,
			"0x%08X - 0x%08X | %12d bytes | SquashFS filesystem\n",
			fsStart,
			fsEnd,
			fsEnd-fsStart+1,
		)
	}

	trailingStart := fsEnd + 1
	trailingEnd := fileSize - 1
	if trailingEnd >= trailingStart {
		fmt.Fprintf(
			file,
			"0x%08X - 0x%08X | %12d bytes | trailing area / padding / footer region\n",
			trailingStart,
			trailingEnd,
			trailingEnd-trailingStart+1,
		)
	}

	return nil
}
