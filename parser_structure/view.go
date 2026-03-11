package main

import "fmt"

func printHexPreview(data []byte) {
	fmt.Printf("------ HEX PREVIEW (first %d bytes)\n", len(data))
	fmt.Println("Offset(h)   Bytes                                             ASCII")

	for i := 0; i < len(data); i += 16 {
		lineEnd := i + 16
		if lineEnd > len(data) {
			lineEnd = len(data)
		}

		fmt.Printf("%08X   ", i)

		for j := i; j < i+16; j++ {
			if j < lineEnd {
				fmt.Printf("%02X ", data[j])
			} else {
				fmt.Printf("   ")
			}
		}

		fmt.Printf("  ")
		for j := i; j < lineEnd; j++ {
			fmt.Printf("%c", asciiForPreview(data[j]))
		}
		fmt.Println()
	}

	fmt.Println()
}

func printTLVTable(data []byte) {
	fmt.Println("----- TLV TABLE")
	fmt.Println("Offset(h)   Length(hex)   Length(dec)   Text")

	found := false

	for i := 0; i+4 <= len(data); i++ {
		field, ok := parseStrictTLVField(data, i)
		if !ok {
			continue
		}

		fmt.Printf("%08X   %02X           %-11d %q\n",
			field.Offset,
			field.Length,
			field.Length,
			field.Text)

		found = true
		i = field.TextEnd - 1
	}

	if !found {
		fmt.Println("no strict TLV fields found")
	}

	fmt.Println()
}

func printASCIIRuns(data []byte, startOffset int, minLen int) {
	fmt.Println("----- EMBEDDED ASCII FRAGMENTS")
	fmt.Println("Offset(h)   Length(dec)   Text")

	inRun := false
	runStart := 0

	for i := 0; i < len(data); i++ {
		b := data[i]

		if isAllowedASCII(b) && b != 0x00 && b != '\t' && b != '\n' && b != '\r' {
			if !inRun {
				inRun = true
				runStart = i
			}
		} else {
			if inRun {
				runLen := i - runStart
				if runLen >= minLen {
					fmt.Printf("%08X   %-11d %q\n",
						startOffset+runStart,
						runLen,
						string(data[runStart:i]))
				}
				inRun = false
			}
		}
	}

	if inRun {
		runLen := len(data) - runStart
		if runLen >= minLen {
			fmt.Printf("%08X   %-11d %q\n",
				startOffset+runStart,
				runLen,
				string(data[runStart:]))
		}
	}

	fmt.Println()
}
