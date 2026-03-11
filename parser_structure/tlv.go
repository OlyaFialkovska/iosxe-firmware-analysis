package main

type TLVField struct {
	Offset    int
	Length    int
	Text      string
	TextStart int
	TextEnd   int
}

func parseStrictTLVField(data []byte, offset int) (TLVField, bool) {
	var field TLVField

	if offset+4 > len(data) {
		return field, false
	}

	if data[offset] != 0x00 || data[offset+1] != 0x00 || data[offset+2] != 0x00 {
		return field, false
	}

	length := int(data[offset+3])
	if length <= 0 {
		return field, false
	}

	textStart := offset + 4
	textEnd := textStart + length

	if textEnd > len(data) {
		return field, false
	}

	rawField := data[textStart:textEnd]

	if !allAllowedASCII(rawField) {
		return field, false
	}

	if !hasLetter(rawField) {
		return field, false
	}

	field = TLVField{
		Offset:    offset,
		Length:    length,
		Text:      printableText(rawField),
		TextStart: textStart,
		TextEnd:   textEnd,
	}

	return field, true
}
