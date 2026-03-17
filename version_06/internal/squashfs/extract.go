package squashfs

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"fwparse/internal/model"
	"fwparse/internal/pkgformat"
	"fwparse/internal/report"
)

func FindSquashfsOffset(reader io.ReaderAt, fileSize int64, start int64) (int64, error) {
	bufSize := int64(1024 * 1024)

	for offset := start; offset < fileSize; offset += bufSize {
		remaining := fileSize - offset
		size := bufSize
		if remaining < size {
			size = remaining
		}

		buf := make([]byte, size)
		_, err := reader.ReadAt(buf, offset)
		if err != nil && err != io.EOF {
			return 0, err
		}

		index := bytes.Index(buf, []byte("hsqs"))
		if index != -1 {
			return offset + int64(index), nil
		}
	}

	return 0, fmt.Errorf("squashfs not found")
}

func LocateFilesystemStructurally(reader io.ReaderAt, fileSize int64, pkgOffset int64) (int64, *model.SquashfsSuperblock, error) {
	payloadOffset, err := pkgformat.LocatePkgPayloadOffset(reader, fileSize, pkgOffset)
	if err != nil {
		return 0, nil, err
	}

	sb, err := ReadSquashfsSuperblock(reader, payloadOffset)
	if err == nil && sb.BytesUsed > 0 {
		return payloadOffset, sb, nil
	}

	const searchWindow = 128 * 1024
	const step = 4096

	limit := payloadOffset + searchWindow
	if limit > fileSize {
		limit = fileSize
	}

	for off := payloadOffset; off+96 <= limit; off += step {
		sb, err := ReadSquashfsSuperblock(reader, off)
		if err == nil && sb.BytesUsed > 0 {
			return off, sb, nil
		}
	}

	return 0, nil, fmt.Errorf("no squashfs found near pkg payload")
}

func SaveExactFilesystem(reader io.ReaderAt, fileSize int64, offset int64, index int) {
	sb, err := ReadSquashfsSuperblock(reader, offset)
	if err != nil {
		fmt.Println("Error reading squashfs superblock for exact extraction:", err)
		return
	}

	size := int64(sb.BytesUsed)
	if size <= 0 {
		fmt.Println("Invalid squashfs size at offset", offset)
		return
	}

	if offset+size > fileSize {
		size = fileSize - offset
	}

	outDir := "output/final_filesystems"
	err = os.MkdirAll(outDir, 0755)
	if err != nil {
		fmt.Println("Error creating final_filesystems folder:", err)
		return
	}

	fileName := fmt.Sprintf("filesystem_%d_0x%X.squashfs", index, offset)
	fullPath := filepath.Join(outDir, fileName)

	out, err := os.Create(fullPath)
	if err != nil {
		fmt.Println("Error creating exact filesystem file:", err)
		return
	}
	defer out.Close()

	buf := make([]byte, 8192)
	current := offset
	remaining := size

	for remaining > 0 {
		toRead := int64(len(buf))
		if toRead > remaining {
			toRead = remaining
		}

		n, err := reader.ReadAt(buf[:toRead], current)
		if err != nil && err != io.EOF {
			fmt.Println("Error reading filesystem bytes:", err)
			return
		}

		_, writeErr := out.Write(buf[:n])
		if writeErr != nil {
			fmt.Println("Error writing filesystem bytes:", writeErr)
			return
		}

		current += int64(n)
		remaining -= int64(n)

		if err == io.EOF {
			break
		}
	}

	fmt.Println("Saved exact filesystem:", fullPath)

	err = report.SaveSquashfsSuperblockReport("output/squashfs_info/final_superblock.txt", offset, sb)
	if err != nil {
		fmt.Println("Error saving squashfs report:", err)
	}
}

func SaveExactFilesystemAt(reader io.ReaderAt, fileSize int64, offset int64, outPath string) error {
	file, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer file.Close()

	buf := make([]byte, 1024*1024)
	current := offset

	for current < fileSize {
		n, err := reader.ReadAt(buf, current)
		if n > 0 {
			_, wErr := file.Write(buf[:n])
			if wErr != nil {
				return wErr
			}
			current += int64(n)
		}

		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
	}

	return nil
}

func ExtractSquashfsToDir(squashfsPath string, outDir string) error {
	err := os.RemoveAll(outDir)
	if err != nil {
		return err
	}

	err = os.MkdirAll(outDir, 0755)
	if err != nil {
		return err
	}

	cmd := exec.Command("unsquashfs", "-f", "-no-xattrs", "-d", outDir, squashfsPath)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("unsquashfs failed: %v\n%s%s", err, stdout.String(), stderr.String())
	}

	return nil
}
