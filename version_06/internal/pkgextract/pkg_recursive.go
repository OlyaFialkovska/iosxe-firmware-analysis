package pkgextract

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"fwparse/internal/squashfs"
)

func ProcessPkgFiles(rootDir string) error {
	return filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() && filepath.Ext(path) == ".pkg" {
			fmt.Println("Processing PKG:", path)

			err := processSinglePkg(path)
			if err != nil {
				fmt.Println("Failed to process:", path, "error:", err)
			}
		}

		return nil
	})
}

func processSinglePkg(pkgPath string) error {
	file, err := os.Open(pkgPath)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	fsOffset, err := squashfs.FindSquashfsOffset(file, info.Size(), 0)
	if err != nil {
		return fmt.Errorf("no squashfs found in pkg")
	}

	fmt.Printf("  -> SquashFS at 0x%X\n", fsOffset)

	sb, err := squashfs.ReadSquashfsSuperblock(file, fsOffset)
	if err != nil {
		return err
	}

	fmt.Printf("  -> SquashFS size: %d bytes\n", sb.BytesUsed)
	fmt.Printf("  -> SquashFS version: %d.%d\n", sb.VersionMajor, sb.VersionMinor)

	outFile := pkgPath + ".squashfs"

	err = squashfs.SaveExactFilesystemAt(file, info.Size(), fsOffset, outFile)
	if err != nil {
		return err
	}

	base := filepath.Base(pkgPath)
	outDir := filepath.Join(filepath.Dir(pkgPath), base+"_rootfs")

	if _, err := os.Stat(outDir); err == nil {
		err = os.RemoveAll(outDir)
		if err != nil {
			return fmt.Errorf("failed to clean existing dir: %w", err)
		}
	}

	err = squashfs.ExtractSquashfsToDir(outFile, outDir)
	if err != nil {
		return err
	}

	fmt.Println("  -> Extracted to:", outDir)

	return nil
}
