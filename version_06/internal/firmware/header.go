package firmware

import (
	"bytes"
	"fmt"
	"io"
	"sort"

	"fwparse/internal/model"
)

func ReadHeader(reader io.ReaderAt, size int) ([]byte, error) {
	if size <= 0 {
		return nil, fmt.Errorf("invalid header size")
	}

	header := make([]byte, size)

	n, err := reader.ReadAt(header, 0)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return header[:n], nil
}

func DefaultHeaderPatterns() []string {
	return []string{
		"KEY_TLV_PACKAGE_COMPATIBILITY",
		"KEY_TLV_PACKAGE_BOOTARCH",
		"KEY_TLV_BOARD_COMPAT",
		"KEY_TLV_CRYPTO_KEYSTRING",
		"CW_BEGIN",
		"CW_FAMILY",
		"CW_IMAGE",
		"CW_VERSION",
		"CW_FULL_VERSION",
		"CW_DESCRIPTION",
		"CW_END",
	}
}

func FindHeaderSections(header []byte, patterns []string) []model.Section {
	var sections []model.Section

	for _, p := range patterns {
		offset := bytes.Index(header, []byte(p))
		if offset == -1 {
			continue
		}

		sections = append(sections, model.Section{
			Name:   p,
			Offset: offset,
			Size:   len(p),
		})
	}

	sort.Slice(sections, func(i, j int) bool {
		return sections[i].Offset < sections[j].Offset
	})

	return sections
}

func BuildSectionRanges(header []byte, sections []model.Section) []string {
	var ranges []string

	if len(sections) == 0 {
		return ranges
	}

	for i := 0; i < len(sections); i++ {
		start := sections[i].Offset

		var end int
		if i < len(sections)-1 {
			end = sections[i+1].Offset - 1
		} else {
			end = len(header) - 1
		}

		rangeSize := end - start + 1
		line := fmt.Sprintf("%s -> 0x%X - 0x%X (%d bytes)", sections[i].Name, start, end, rangeSize)
		ranges = append(ranges, line)
	}

	return ranges
}

func CheckSectionBoundaries(header []byte, sections []model.Section) []string {
	var errors []string
	headerSize := len(header)

	for i := 0; i < len(sections); i++ {
		s := sections[i]

		if s.Offset < 0 || s.Offset >= headerSize {
			errors = append(errors,
				fmt.Sprintf("Boundary error: %s has invalid offset 0x%X", s.Name, s.Offset))
		}

		if s.Size < 0 {
			errors = append(errors,
				fmt.Sprintf("Boundary error: %s has negative size %d", s.Name, s.Size))
		}

		sectionEnd := s.Offset + s.Size - 1
		if sectionEnd >= headerSize {
			errors = append(errors,
				fmt.Sprintf("Boundary error: %s goes outside header (end: 0x%X)", s.Name, sectionEnd))
		}

		if i < len(sections)-1 {
			next := sections[i+1]
			if s.Offset+s.Size > next.Offset {
				errors = append(errors,
					fmt.Sprintf("Overlap error: %s overlaps with %s", s.Name, next.Name))
			}
		}
	}

	return errors
}
