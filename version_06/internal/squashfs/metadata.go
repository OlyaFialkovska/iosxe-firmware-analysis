package squashfs

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/ulikunitz/xz"
)

func TryDecompressXZ(data []byte) ([]byte, bool) {
	r, err := xz.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, false
	}

	out, err := io.ReadAll(r)
	if err != nil {
		return nil, false
	}

	return out, true
}

func ReadMetadataBlock(reader io.ReaderAt, offset int64) ([]byte, int64, string, error) {
	header := make([]byte, 2)

	_, err := reader.ReadAt(header, offset)
	if err != nil {
		return nil, offset, "", err
	}

	rawHeader := binary.LittleEndian.Uint16(header)
	blockSize := int64(rawHeader & 0x7FFF)

	if blockSize <= 0 {
		return nil, offset, "", fmt.Errorf("invalid metadata block size")
	}

	blockData := make([]byte, blockSize)
	_, err = reader.ReadAt(blockData, offset+2)
	if err != nil {
		return nil, offset, "", err
	}

	if decoded, ok := TryDecompressXZ(blockData); ok {
		return decoded, offset + 2 + blockSize, "xz", nil
	}

	return blockData, offset + 2 + blockSize, "raw", nil
}

func ExtractASCIIStrings(data []byte, minLen int) []string {
	var result []string
	var current []byte

	for _, b := range data {
		if b >= 32 && b <= 126 {
			current = append(current, b)
		} else {
			if len(current) >= minLen {
				result = append(result, string(current))
			}
			current = nil
		}
	}

	if len(current) >= minLen {
		result = append(result, string(current))
	}

	return result
}

func SaveMetadataStrings(data []byte, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	fmt.Fprintln(file, "Metadata Strings")
	fmt.Fprintln(file, "---")

	strs := ExtractASCIIStrings(data, 4)
	if len(strs) == 0 {
		fmt.Fprintln(file, "No printable strings found")
		return nil
	}

	for _, s := range strs {
		fmt.Fprintln(file, s)
	}

	return nil
}
