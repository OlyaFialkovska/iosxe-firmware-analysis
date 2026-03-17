package app

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"fwparse/internal/debug"
	"fwparse/internal/firmware"
	"fwparse/internal/model"
	"fwparse/internal/pkgextract"
	"fwparse/internal/pkgformat"
	"fwparse/internal/report"
	"fwparse/internal/squashfs"
	"fwparse/internal/util"
)

func Run() error {
	if len(os.Args) < 2 {
		fmt.Println("Usage: fwparse <firmware.bin>")
		return fmt.Errorf("no input file")
	}

	filePath := os.Args[1]

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("error reading file info: %w", err)
	}

	fmt.Println("Firmware size:", info.Size(), "bytes")

	var reader io.ReaderAt = file

	header, err := firmware.ReadHeader(reader, 4096)
	if err != nil {
		return fmt.Errorf("error reading header: %w", err)
	}

	fwHeader := model.FirmwareHeader{
		Data: header,
	}

	fmt.Println("First 4KB of firmware read successfully")

	fmt.Println("First 64 bytes of header:")
	limit := 64
	if len(fwHeader.Data) < limit {
		limit = len(fwHeader.Data)
	}

	for i := 0; i < limit; i++ {
		fmt.Printf("%02X ", fwHeader.Data[i])
		if (i+1)%16 == 0 {
			fmt.Println()
		}
	}

	fmt.Println("\nSearching for metadata sections...")

	patterns := firmware.DefaultHeaderPatterns()
	sections := firmware.FindHeaderSections(fwHeader.Data, patterns)
	ranges := firmware.BuildSectionRanges(fwHeader.Data, sections)
	boundaryErrors := firmware.CheckSectionBoundaries(fwHeader.Data, sections)

	fmt.Println("\nFound sections:")
	for _, s := range sections {
		fmt.Printf("Section: %s | Offset: 0x%X | Size: %d bytes\n", s.Name, s.Offset, s.Size)
	}

	fmt.Println("\nApproximate section ranges:")
	for _, r := range ranges {
		fmt.Println(r)
	}

	fmt.Println("\nChecking section boundaries...")
	if len(boundaryErrors) == 0 {
		fmt.Println("All section boundary checks passed")
	} else {
		for _, e := range boundaryErrors {
			fmt.Println(e)
		}
	}

	err = os.MkdirAll("output", 0755)
	if err != nil {
		return fmt.Errorf("error creating output folder: %w", err)
	}

	err = os.MkdirAll("output/squashfs_info", 0755)
	if err != nil {
		return fmt.Errorf("error creating squashfs_info folder: %w", err)
	}

	err = report.SaveHeaderReport(
		"output/header_sections.txt",
		info.Size(),
		len(fwHeader.Data),
		sections,
		ranges,
		boundaryErrors,
	)
	if err != nil {
		return fmt.Errorf("error saving header report: %w", err)
	}

	fmt.Println("\nSection map saved to output/header_sections.txt")

	meta, err := firmware.ParseTopLevelMetadata(fwHeader.Data)
	if err != nil {
		fmt.Println("\nCould not parse metadata values")
		return nil
	}

	fmt.Println("\nParsed metadata values:")
	fmt.Println("CW_FAMILY =", meta.Family)
	fmt.Println("CW_IMAGE =", meta.Image)
	fmt.Println("CW_VERSION =", meta.Version)
	fmt.Println("CW_FULL_VERSION =", meta.FullVersion)
	fmt.Println("CW_DESCRIPTION =", meta.Description)

	err = report.SaveMetadataReport("output/metadata_values.txt", meta)
	if err != nil {
		return fmt.Errorf("error saving metadata values: %w", err)
	}

	fmt.Println("Metadata values saved to output/metadata_values.txt")

	nextComponentOffset := int64(len(fwHeader.Data))
	nextComponentSize := 8192

	remainingBytes := info.Size() - nextComponentOffset
	if remainingBytes <= 0 {
		return fmt.Errorf("no bytes left for next component")
	}

	if int64(nextComponentSize) > remainingBytes {
		nextComponentSize = int(remainingBytes)
	}

	root := &model.Node{
		Name:   "firmware.bin",
		Offset: 0,
		Size:   int(info.Size()),
		Type:   "firmware container",
	}

	metaNode := &model.Node{
		Name:   "top-level metadata",
		Offset: int64(meta.BeginOffset),
		Size:   meta.EndOffset + len("CW_END") - meta.BeginOffset,
		Type:   "header metadata",
		Details: fmt.Sprintf(
			"family=%s, image=%s, version=%s",
			meta.Family,
			meta.Image,
			meta.Version,
		),
	}

	root.Children = append(root.Children, metaNode)

	nextNode := firmware.InspectComponent(reader, info.Size(), nextComponentOffset, nextComponentSize, 1)
	root.Children = append(root.Children, nextNode)

	fmt.Println("Firmware structure preview added")

	pkgOffset := nextComponentOffset

	previewSize := 512
	remainingPreview := info.Size() - pkgOffset
	if remainingPreview > 0 {
		if int64(previewSize) > remainingPreview {
			previewSize = int(remainingPreview)
		}

		buf := make([]byte, previewSize)
		n, err := reader.ReadAt(buf, pkgOffset)
		if err == nil || err == io.EOF {
			buf = buf[:n]

			fmt.Printf("\nBytes after top-level metadata at 0x%X:\n", pkgOffset)
			for i := 0; i < len(buf) && i < 128; i++ {
				fmt.Printf("%02X ", buf[i])
				if (i+1)%16 == 0 {
					fmt.Println()
				}
			}
			fmt.Println()

			fmt.Println("\nASCII strings near next component:")
			strs := util.ExtractASCIIStrings(buf, 4)
			for _, s := range strs {
				fmt.Println(s)
			}
		}
	}

	pkgPreview := make([]byte, 4096)
	n, readErr := reader.ReadAt(pkgPreview, pkgOffset)
	if readErr != nil && readErr != io.EOF {
		return fmt.Errorf("error reading pkg preview: %w", readErr)
	}
	pkgPreview = pkgPreview[:n]

	if bytes.Contains(pkgPreview, []byte("CW_BEGIN")) {
		fmt.Println("Nested metadata was found near pkg offset")
	}

	pkgNode := pkgformat.AnalyzePkgHeader(reader, info.Size(), pkgOffset)
	root.Children = append(root.Children, pkgNode)

	err = report.SaveFirmwareTreeReport("output/firmware_structure.txt", root, util.WriteTree)
	if err != nil {
		return fmt.Errorf("error saving firmware structure: %w", err)
	}

	fmt.Println("Firmware structure saved to output/firmware_structure.txt")

	debug.SavePkgOffsetAnalysis(reader, info.Size(), pkgOffset)

	previewSize = 4096
	remainingPreview = info.Size() - pkgOffset
	if remainingPreview > 0 {
		if int64(previewSize) > remainingPreview {
			previewSize = int(remainingPreview)
		}

		pkgData := make([]byte, previewSize)
		n, err := reader.ReadAt(pkgData, pkgOffset)
		if err == nil || err == io.EOF {
			pkgData = pkgData[:n]

			start, mode, parsedFields := pkgformat.FindBestPkgFieldStart(pkgData)
			if start != -1 && len(parsedFields) > 0 {
				debug.SavePkgParsedFields(parsedFields, pkgOffset)

				modeFile, err := os.Create("output/pkg_field_parse_mode.txt")
				if err == nil {
					defer modeFile.Close()

					fmt.Fprintln(modeFile, "PKG Field Parse Mode")
					fmt.Fprintln(modeFile, "---")
					fmt.Fprintf(modeFile, "Chosen start offset: 0x%X\n", start)
					fmt.Fprintf(modeFile, "Chosen length mode: %s\n", mode)
					fmt.Fprintf(modeFile, "Parsed fields: %d\n", len(parsedFields))
				}
			}
		}
	}

	fsOffset, err := squashfs.FindSquashfsOffset(reader, info.Size(), nextComponentOffset)
	if err != nil {
		return fmt.Errorf("error finding squashfs: %w", err)
	}

	fmt.Printf("SquashFS found at 0x%X\n", fsOffset)

	sb, err := squashfs.ReadSquashfsSuperblock(reader, fsOffset)
	if err != nil {
		return fmt.Errorf("error reading squashfs superblock: %w", err)
	}

	fmt.Printf("SquashFS bytes used: %d\n", sb.BytesUsed)

	squashfsPath := "output/rootfs.squashfs"

	err = squashfs.SaveExactFilesystemAt(reader, info.Size(), fsOffset, squashfsPath)
	if err != nil {
		return fmt.Errorf("error saving exact filesystem: %w", err)
	}
	extractDir := "output/extracted_rootfs"
	err = squashfs.ExtractSquashfsToDir(squashfsPath, extractDir)
	if err != nil {
		return fmt.Errorf("error extracting squashfs tree: %w", err)
	}

	err = squashfs.SaveDirectoryTree(extractDir, "output/filesystem_tree.txt")
	if err != nil {
		return fmt.Errorf("error saving filesystem tree: %w", err)
	}

	err = squashfs.SaveExtractedFileInventory(extractDir, "output/file_inventory.txt")
	if err != nil {
		return fmt.Errorf("error saving file inventory: %w", err)
	}

	err = report.SaveSquashfsSuperblockReport("output/squashfs_info/final_superblock.txt", fsOffset, sb)
	if err != nil {
		return fmt.Errorf("error saving squashfs superblock report: %w", err)
	}

	err = report.SaveFilesystemLocationReport(
		"output/filesystem_location.txt",
		filePath,
		info.Size(),
		fsOffset,
		sb,
	)
	if err != nil {
		return fmt.Errorf("error saving filesystem location report: %w", err)
	}

	err = report.SaveByteRangesReport(
		"output/byte_ranges.txt",
		info.Size(),
		int64(len(fwHeader.Data)),
		fsOffset,
		int64(sb.BytesUsed),
	)
	if err != nil {
		return fmt.Errorf("error saving byte ranges report: %w", err)
	}

	fmt.Println("Filesystem extracted successfully")
	fmt.Println("Tree saved to output/filesystem_tree.txt")
	fmt.Println("Inventory saved to output/file_inventory.txt")
	fmt.Println("Filesystem location saved to output/filesystem_location.txt")
	fmt.Println("Byte ranges saved to output/byte_ranges.txt")

	fmt.Println("\nStarting recursive PKG extraction...")

	err = pkgextract.ProcessPkgFiles("output/extracted_rootfs")
	if err != nil {
		return fmt.Errorf("error processing pkg files: %w", err)
	}

	fmt.Println("PKG recursive extraction completed")

	fmt.Println("\nBuilding unified filesystem tree...")

	err = pkgextract.BuildUnifiedTree("output/extracted_rootfs", "output/final_filesystem_tree.txt")
	if err != nil {
		return fmt.Errorf("error building unified tree: %w", err)
	}

	fmt.Println("Unified filesystem tree saved to: output/final_filesystem_tree.txt")

	err = pkgextract.BuildUnifiedInventory("output/extracted_rootfs", "output/final_file_inventory.txt")
	if err != nil {
		return fmt.Errorf("error building unified inventory: %w", err)
	}

	fmt.Println("Unified file inventory saved to: output/final_file_inventory.txt")

	return nil
}
