package firmware

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"fwparse/internal/model"
)

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

func detectComponentType(data []byte, asciiStrings []string) string {
	if len(data) >= 4 {
		if bytes.Equal(data[:4], []byte{0x7F, 0x45, 0x4C, 0x46}) {
			return "ELF executable"
		}
		if data[0] == 0x1F && data[1] == 0x8B {
			return "GZIP compressed data"
		}
		if bytes.Equal(data[:4], []byte("hsqs")) {
			return "SquashFS filesystem"
		}
		if bytes.Equal(data[:4], []byte("PK\x03\x04")) {
			return "ZIP archive"
		}
	}

	for _, s := range asciiStrings {
		if strings.Contains(s, "Cisco") || strings.Contains(s, "IOS-XE") {
			return "Certificate or Cisco metadata block"
		}
		if strings.Contains(s, "boot") || strings.Contains(s, "reboot") {
			return "Possible bootloader code/data"
		}
		if strings.Contains(s, ".pkg") {
			return "Possible package-related component"
		}
	}

	return "Unknown binary component"
}

func InspectComponent(reader io.ReaderAt, fileSize int64, offset int64, size int, depth int) *model.Node {
	if depth > 4 {
		return &model.Node{
			Name:   "max_depth_reached",
			Offset: offset,
			Size:   size,
			Type:   "stop",
		}
	}

	if offset >= fileSize {
		return &model.Node{
			Name:   "invalid_component",
			Offset: offset,
			Size:   0,
			Type:   "invalid",
		}
	}

	remaining := fileSize - offset
	if int64(size) > remaining {
		size = int(remaining)
	}
	if size < 0 {
		size = 0
	}

	data := make([]byte, size)
	n, err := reader.ReadAt(data, offset)
	if err != nil && err != io.EOF {
		return &model.Node{
			Name:    "read_error_component",
			Offset:  offset,
			Size:    0,
			Type:    "error",
			Details: "read error",
		}
	}

	data = data[:n]
	strs := extractASCIIStrings(data, 4)
	componentType := detectComponentType(data, strs)

	node := &model.Node{
		Name:   "component",
		Offset: offset,
		Size:   len(data),
		Type:   componentType,
	}

	beginOffset := bytes.Index(data, []byte("CW_BEGIN"))
	endOffset := bytes.Index(data, []byte("CW_END"))

	if beginOffset != -1 && endOffset != -1 && endOffset > beginOffset {
		metaText := string(data[beginOffset : endOffset+len("CW_END")])

		image := extractValue(metaText, "CW_IMAGE")
		version := extractValue(metaText, "CW_VERSION")
		description := extractValue(metaText, "CW_DESCRIPTION")

		childName := "metadata block"
		childType := "nested metadata"

		if image != "" {
			childName = image
		}
		if strings.Contains(image, ".pkg") {
			childType = "pkg-like metadata"
		}

		metaNode := &model.Node{
			Name:    childName,
			Offset:  offset + int64(beginOffset),
			Size:    endOffset + len("CW_END") - beginOffset,
			Type:    childType,
			Details: fmt.Sprintf("version=%s, description=%s", version, description),
		}

		node.Children = append(node.Children, metaNode)
	}

	return node
}
