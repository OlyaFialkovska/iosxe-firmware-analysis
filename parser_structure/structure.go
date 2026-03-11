package main

import "fmt"

func parseStructure(data []byte) {
	fmt.Println("----- STRUCTURE SUMMARY")

	inMetadata := false
	inCW := false
	afterCW := false
	lastTLVEnd := -1

	for i := 0; i+4 <= len(data); i++ {
		field, ok := parseStrictTLVField(data, i)
		if ok {
			if !inMetadata {
				fmt.Printf("[0x%08X]  METADATA_REGION_START\n\n", field.Offset)
				inMetadata = true
			}

			if !inCW && !afterCW && startsWith(field.Text, "CW_BEGIN") {
				fmt.Printf("[0x%08X]  CW_SUBBLOCK_START\n", field.Offset)
				fmt.Printf("             length: %d\n", field.Length)
				fmt.Printf("             text  : %q\n\n", field.Text)
				inCW = true
			} else if inCW && startsWith(field.Text, "CW_END") {
				fmt.Printf("[0x%08X]  CW_SUBBLOCK_END\n", field.Offset)
				fmt.Printf("             length: %d\n", field.Length)
				fmt.Printf("             text  : %q\n\n", field.Text)
				inCW = false
				afterCW = true
			} else if inCW {
				fmt.Printf("[0x%08X]  CW_FIELD\n", field.Offset)
				fmt.Printf("             length: %d\n", field.Length)
				fmt.Printf("             text  : %q\n\n", field.Text)
			} else if afterCW {
				fmt.Printf("[0x%08X]  POST_CW_METADATA_FIELD\n", field.Offset)
				fmt.Printf("             length: %d\n", field.Length)
				fmt.Printf("             text  : %q\n\n", field.Text)
			} else {
				fmt.Printf("[0x%08X]  METADATA_FIELD\n", field.Offset)
				fmt.Printf("             length: %d\n", field.Length)
				fmt.Printf("             text  : %q\n\n", field.Text)
			}

			lastTLVEnd = field.TextEnd
			i = field.TextEnd - 1
			continue
		}

		if inMetadata && lastTLVEnd != -1 {
			if i-lastTLVEnd > metadataGapLimit {
				fmt.Printf("[0x%08X]  APPROX_METADATA_REGION_END\n\n", lastTLVEnd)
				return
			}
		}
	}

	if lastTLVEnd == -1 {
		fmt.Println("no TLV-like structure detected")
	} else {
		fmt.Printf("[0x%08X]  APPROX_METADATA_REGION_END (preview ended)\n\n", lastTLVEnd)
	}

	fmt.Println()
}
