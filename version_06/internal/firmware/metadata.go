package firmware

import (
	"bytes"
	"fmt"
	"strings"
)

type Metadata struct {
	BeginOffset int
	EndOffset   int

	Family      string
	Image       string
	Version     string
	FullVersion string
	Description string
}

func extractValue(block string, key string) string {
	start := strings.Index(block, key+"=$")
	if start == -1 {
		return ""
	}

	start = start + len(key) + 2

	end := strings.Index(block[start:], "$")
	if end == -1 {
		return ""
	}

	return block[start : start+end]
}

func ParseTopLevelMetadata(data []byte) (*Metadata, error) {
	begin := bytes.Index(data, []byte("CW_BEGIN"))
	end := bytes.Index(data, []byte("CW_END"))

	if begin == -1 || end == -1 || end <= begin {
		return nil, fmt.Errorf("metadata block not found")
	}

	block := data[begin : end+len("CW_END")]
	text := string(block)

	meta := &Metadata{
		BeginOffset: begin,
		EndOffset:   end,
	}

	meta.Family = extractValue(text, "CW_FAMILY")
	meta.Image = extractValue(text, "CW_IMAGE")
	meta.Version = extractValue(text, "CW_VERSION")
	meta.FullVersion = extractValue(text, "CW_FULL_VERSION")
	meta.Description = extractValue(text, "CW_DESCRIPTION")

	return meta, nil
}
