package squashfs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func detectRealType(info os.FileInfo) string {
	mode := info.Mode()

	if mode.IsDir() {
		return "directory"
	}
	if mode.IsRegular() {
		return "file"
	}
	if mode&os.ModeSymlink != 0 {
		return "symlink"
	}
	if mode&os.ModeDevice != 0 {
		return "device"
	}
	if mode&os.ModeNamedPipe != 0 {
		return "pipe"
	}
	if mode&os.ModeSocket != 0 {
		return "socket"
	}

	return "other"
}

func SaveExtractedFileInventory(rootDir string, outPath string) error {
	file, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer file.Close()

	fmt.Fprintln(file, "Extracted File Inventory")
	fmt.Fprintln(file, "---")

	return filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}

		if rel == "." {
			rel = "/"
		} else {
			rel = "/" + filepath.ToSlash(rel)
		}

		fmt.Fprintf(
			file,
			"path=%s | size=%d | type=%s\n",
			rel,
			info.Size(),
			detectRealType(info),
		)

		return nil
	})
}

func SaveDirectoryTree(rootDir string, outPath string) error {
	file, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer file.Close()

	fmt.Fprintln(file, "Filesystem Tree")
	fmt.Fprintln(file, "---")

	return filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}

		level := 0
		if rel != "." {
			level = strings.Count(filepath.ToSlash(rel), "/") + 1
		}

		prefix := strings.Repeat("  ", level)

		name := info.Name()
		if rel == "." {
			name = "/"
		}

		fmt.Fprintf(
			file,
			"%s- %s | size=%d | type=%s\n",
			prefix,
			name,
			info.Size(),
			detectRealType(info),
		)

		return nil
	})
}
