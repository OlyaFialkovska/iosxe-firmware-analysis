package pkgformat

import (
	"fmt"
	"io"
	"os"
	"strings"

	"fwparse/internal/model"
)

func AnalyzePkgHeader(reader io.ReaderAt, fileSize int64, offset int64) *model.Node {
	headerSize := 4096

	remaining := fileSize - offset
	if remaining <= 0 {
		return &model.Node{
			Name:    "pkg_header",
			Offset:  offset,
			Size:    0,
			Type:    "invalid",
			Details: "no bytes left",
		}
	}

	if int64(headerSize) > remaining {
		headerSize = int(remaining)
	}

	data := make([]byte, headerSize)
	n, err := reader.ReadAt(data, offset)
	if err != nil && err != io.EOF {
		return &model.Node{
			Name:    "pkg_header",
			Offset:  offset,
			Size:    0,
			Type:    "error",
			Details: "read error",
		}
	}

	data = data[:n]

	node := &model.Node{
		Name:   "pkg_header",
		Offset: offset,
		Size:   len(data),
		Type:   "pkg header analysis",
	}

	text := string(data)

	fields := []string{
		"CW_BEGIN",
		"CW_FAMILY",
		"CW_IMAGE",
		"CW_VERSION",
		"CW_FULL_VERSION",
		"CW_DESCRIPTION",
		"CW_END",
		"KEY_TLV_PACKAGE_COMPATIBILITY",
		"KEY_TLV_PACKAGE_BOOTARCH",
		"KEY_TLV_BOARD_COMPAT",
		"KEY_TLV_CRYPTO_KEYSTRING",
	}

	var found []string
	for _, f := range fields {
		if strings.Contains(text, f) {
			found = append(found, f)
		}
	}

	start, mode, parsedFields := FindBestPkgFieldStart(data)
	if start != -1 && len(parsedFields) > 0 {
		node.Details = fmt.Sprintf(
			"parsed %d pkg binary fields starting at local offset 0x%X using %s length",
			len(parsedFields),
			start,
			mode,
		)

		for i, f := range parsedFields {
			fieldNode := &model.Node{
				Name:    fmt.Sprintf("pkg_field_%d", i+1),
				Offset:  offset + int64(f.Offset),
				Size:    f.Length,
				Type:    "pkg binary field",
				Details: fmt.Sprintf("tag=0x%02X, length=%d", f.Tag, f.Length),
			}

			node.Children = append(node.Children, fieldNode)
		}

		modeFile, err := os.Create("../../output/pkg_field_parse_mode.txt")
		if err == nil {
			defer modeFile.Close()

			fmt.Fprintln(modeFile, "PKG Field Parse Mode")
			fmt.Fprintln(modeFile, "---")
			fmt.Fprintf(modeFile, "Chosen start offset: 0x%X\n", start)
			fmt.Fprintf(modeFile, "Chosen length mode: %s\n", mode)
			fmt.Fprintf(modeFile, "Parsed fields: %d\n", len(parsedFields))
		}
	}

	if node.Details == "" {
		if len(found) > 0 {
			node.Details = "found metadata-like fields: " + strings.Join(found, ", ")
		} else {
			node.Details = "no known CW/KEY_TLV fields found in first 4KB"
		}
	}

	return node
}
