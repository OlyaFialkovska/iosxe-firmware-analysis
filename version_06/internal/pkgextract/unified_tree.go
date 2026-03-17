package pkgextract

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func BuildUnifiedTree(baseDir string, outFile string) error {
	file, err := os.Create(outFile)
	if err != nil {
		return err
	}
	defer file.Close()

	fmt.Fprintln(file, "Unified Filesystem Tree")
	fmt.Fprintln(file, "---")
	fmt.Fprintln(file, "/")

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

		fmt.Fprintf(file, "  - %s | type=directory\n", pkgName)

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
			depth := strings.Count(relPath, "/") + 2
			prefix := strings.Repeat("  ", depth)

			itemType := "file"
			if info.IsDir() {
				itemType = "directory"
			} else if info.Mode()&os.ModeSymlink != 0 {
				itemType = "symlink"
			}

			fmt.Fprintf(
				file,
				"%s- %s | size=%d | type=%s\n",
				prefix,
				info.Name(),
				info.Size(),
				itemType,
			)

			return nil
		})
		if err != nil {
			return err
		}

		fmt.Fprintln(file)
	}

	return nil
}

func extractPkgName(name string) string {
	name = strings.TrimSuffix(name, ".pkg_rootfs")

	parts := strings.Split(name, "-")
	if len(parts) > 1 {
		name = parts[1]
	}

	if idx := strings.Index(name, "."); idx != -1 {
		name = name[:idx]
	}

	return name
}
