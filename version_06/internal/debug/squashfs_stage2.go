package debug

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ulikunitz/xz"

	"fwparse/internal/squashfs"
)

func tryDecompressXZ(data []byte) ([]byte, bool) {
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

func readMetadataBlock(reader io.ReaderAt, offset int64) ([]byte, int64, string, error) {
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

	if decoded, ok := tryDecompressXZ(blockData); ok {
		return decoded, offset + 2 + blockSize, "xz", nil
	}

	return blockData, offset + 2 + blockSize, "raw", nil
}

func extractASCIIStrings(data []byte, minLen int) []string {
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

func saveMetadataStrings(data []byte, filePath string) {
	file, err := os.Create(filePath)
	if err != nil {
		fmt.Println("Error creating metadata strings file:", err)
		return
	}
	defer file.Close()

	fmt.Fprintln(file, "Metadata Strings")
	fmt.Fprintln(file, "---")

	strs := extractASCIIStrings(data, 4)
	if len(strs) == 0 {
		fmt.Fprintln(file, "No printable strings found")
		return
	}

	for _, s := range strs {
		fmt.Fprintln(file, s)
	}
}

func isGoodNameChar(b byte) bool {
	if b >= 'a' && b <= 'z' {
		return true
	}
	if b >= 'A' && b <= 'Z' {
		return true
	}
	if b >= '0' && b <= '9' {
		return true
	}

	switch b {
	case '.', '_', '-', '+', '/':
		return true
	}

	return false
}

func looksLikeFsName(s string) bool {
	if len(s) < 2 || len(s) > 120 {
		return false
	}

	if strings.Contains(s, "Cisco") ||
		strings.Contains(s, "IOS-XE") ||
		strings.Contains(s, "squashfs") ||
		strings.Contains(s, "block") {
		return false
	}

	if strings.HasPrefix(s, ".") && len(s) == 1 {
		return false
	}

	hasLetter := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
			hasLetter = true
			break
		}
	}

	return hasLetter
}

func collectNameCandidates(data []byte) []string {
	var result []string
	var current []byte

	flush := func() {
		if len(current) == 0 {
			return
		}

		s := string(current)
		if looksLikeFsName(s) {
			result = append(result, s)
		}
		current = nil
	}

	for _, b := range data {
		if isGoodNameChar(b) {
			current = append(current, b)
		} else {
			flush()
		}
	}

	flush()

	return result
}

func detectFileType(name string) string {
	name = strings.ToLower(name)

	if strings.HasSuffix(name, ".so") {
		return "shared library"
	}
	if strings.HasSuffix(name, ".conf") {
		return "config"
	}
	if strings.HasSuffix(name, ".sh") {
		return "script"
	}
	if strings.HasSuffix(name, ".bin") {
		return "binary"
	}
	if strings.Contains(name, "busybox") {
		return "ELF executable"
	}

	return "file"
}

func SaveSquashfsStage2(reader io.ReaderAt, offset int64, name string) {
	sb, err := squashfs.ReadSquashfsSuperblock(reader, offset)
	if err != nil {
		fmt.Println("Error reading squashfs superblock:", err)
		return
	}

	baseName := strings.TrimSuffix(name, ".squashfs")
	outDir := "output/squashfs_stage2/" + baseName

	err = os.MkdirAll(outDir, 0755)
	if err != nil {
		fmt.Println("Error creating output folder:", err)
		return
	}

	summaryFile, err := os.Create(outDir + "/metadata_blocks.txt")
	if err != nil {
		fmt.Println("Error creating summary file:", err)
		return
	}
	defer summaryFile.Close()

	fmt.Fprintln(summaryFile, "SquashFS Metadata Analysis")
	fmt.Fprintln(summaryFile, "---")
	fmt.Fprintf(summaryFile, "Offset: 0x%X\n", offset)
	fmt.Fprintf(summaryFile, "Inode table: 0x%X\n", sb.InodeTableStart)
	fmt.Fprintf(summaryFile, "Directory table: 0x%X\n", sb.DirectoryTableStart)
	fmt.Fprintln(summaryFile)

	type tableInfo struct {
		Name  string
		Start uint64
	}

	tables := []tableInfo{
		{Name: "inode_table", Start: sb.InodeTableStart},
		{Name: "directory_table", Start: sb.DirectoryTableStart},
	}

	for _, table := range tables {
		fmt.Fprintf(summaryFile, "%s\n", table.Name)
		fmt.Fprintln(summaryFile, "---")

		current := offset + int64(table.Start)

		for i := 0; i < 3; i++ {
			blockData, nextOffset, mode, err := readMetadataBlock(reader, current)
			if err != nil {
				fmt.Fprintf(summaryFile, "block %d error: %v\n", i+1, err)
				break
			}

			blockPath := fmt.Sprintf("%s/%s_block_%d.bin", outDir, table.Name, i+1)
			err = os.WriteFile(blockPath, blockData, 0644)
			if err != nil {
				fmt.Println("Error saving block:", err)
				break
			}

			stringsPath := fmt.Sprintf("%s/%s_block_%d_strings.txt", outDir, table.Name, i+1)
			saveMetadataStrings(blockData, stringsPath)

			fmt.Fprintf(
				summaryFile,
				"block %d | offset=0x%X | size=%d | mode=%s\n",
				i+1,
				current,
				len(blockData),
				mode,
			)

			if table.Name == "directory_table" {
				names := collectNameCandidates(blockData)
				if len(names) > 0 {
					fmt.Fprintln(summaryFile, "  name candidates:")
					for _, n := range names {
						fmt.Fprintf(summaryFile, "    - %s | guessed_type=%s\n", n, detectFileType(n))
					}
				}
			}

			current = nextOffset
		}

		fmt.Fprintln(summaryFile)
	}

	fmt.Println("Stage2 metadata saved:", outDir)
}
