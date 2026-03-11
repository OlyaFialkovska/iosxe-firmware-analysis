package main

func isAllowedASCII(b byte) bool {
	if b >= 128 {
		return false
	}
	return asciiAllowed[b] == b
}

func hasLetter(data []byte) bool {
	for i := 0; i < len(data); i++ {
		b := data[i]
		if (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') {
			return true
		}
	}
	return false
}

func startsWith(text string, prefix string) bool {
	if len(text) < len(prefix) {
		return false
	}
	return text[:len(prefix)] == prefix
}

func asciiForPreview(b byte) byte {
	if !isAllowedASCII(b) {
		return '.'
	}
	if b == '\t' || b == '\n' || b == '\r' {
		return '.'
	}
	return b
}

func printableText(data []byte) string {
	out := make([]byte, len(data))

	for i := 0; i < len(data); i++ {
		if isAllowedASCII(data[i]) {
			if data[i] == '\t' || data[i] == '\n' || data[i] == '\r' {
				out[i] = ' '
			} else {
				out[i] = data[i]
			}
		} else {
			out[i] = '.'
		}
	}

	return string(out)
}

func allAllowedASCII(data []byte) bool {
	if len(data) == 0 {
		return false
	}

	for i := 0; i < len(data); i++ {
		if !isAllowedASCII(data[i]) {
			return false
		}
	}

	return true
}
