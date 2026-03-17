package pkgextract

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func BuildUnifiedInventory(baseDir string, outFile string) error {
	file, err := os.Create(outFile)
	if err != nil {
		return err
	}
	defer file.Close()

	fmt.Fprintln(file, "Unified File Inventory")
	fmt.Fprintln(file, "---")

	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return err
	}

	var pkgDirs []string
	for _, entry := range entries {
		if entry.IsDir() && strings.HasSuffix(entry.Name(), ".pkg_rootfs") {
			pkgDirs = append(pkgDirs, entry.Name())
		}
	}

	sort.Strings(pkgDirs)

	for _, dirName := range pkgDirs {
		pkgName := extractPkgName(dirName)
		srcRoot := filepath.Join(baseDir, dirName)

		err := filepath.Walk(srcRoot, func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return nil
			}

			relPath, err := filepath.Rel(srcRoot, path)
			if err != nil {
				return nil
			}

			if relPath == "." {
				return nil
			}

			relPath = filepath.ToSlash(relPath)
			fullPath := "/" + pkgName + "/" + relPath

			itemType := "file"
			if info.IsDir() {
				itemType = "directory"
			} else if info.Mode()&os.ModeSymlink != 0 {
				itemType = "symlink"
			}

			fmt.Fprintf(
				file,
				"path=%s | size=%d | type=%s\n",
				fullPath,
				info.Size(),
				itemType,
			)

			return nil
		})
		if err != nil {
			return err
		}
	}

	return nil
}
