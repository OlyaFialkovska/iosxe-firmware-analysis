package main

import (
	"fmt"
	"os"
)

func main() {
	filename := "cat9k-rpbase.16.12.07.SPA.pkg"

	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Println("read error:", err)
		os.Exit(1)
	}

	n := previewSize
	if len(data) < n {
		n = len(data)
	}

	head := data[:n]

	fmt.Println("file:", filename)
	fmt.Println("full size:", len(data), "bytes")
	fmt.Println("preview size:", n, "bytes")
	fmt.Println()

	printHexPreview(head)
	printTLVTable(head)
	printASCIIRuns(head, 0, 4)
	parseStructure(head)
}
