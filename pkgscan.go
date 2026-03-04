package main

import (
	"bytes"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
)

type Sig struct {
	Name string
	Hex  string
	Raw  []byte
}

func findAll(data, needle []byte) []int {
	var offsets []int
	for i := 0; ; {
		j := bytes.Index(data[i:], needle)
		if j == -1 {
			break
		}
		pos := i + j
		offsets = append(offsets, pos)
		i = pos + 1
	}
	return offsets
}

func main() {
	in := flag.String("in", "", "path to .pkg file")
	flag.Parse()
	if *in == "" {
		fmt.Println("Usage: pkgscan -in <file.pkg>")
		os.Exit(1)
	}

	data, err := os.ReadFile(*in)
	if err != nil {
		fmt.Println("read error:", err)
		os.Exit(1)
	}

	sigs := []Sig{
		{Name: "ELF", Hex: "7f454c46"},
		{Name: "GZIP", Hex: "1f8b"},
		{Name: "XZ", Hex: "fd377a585a00"},
		{Name: "CPIO_NEWC_ASCII_070701", Hex: hex.EncodeToString([]byte("070701"))},
		{Name: "SQUASHFS_hsqs", Hex: hex.EncodeToString([]byte("hsqs"))}, // squashfs magic sometimes appears as hsqs in byte view
	}

	// decode hex to raw bytes
	for i := range sigs {
		raw, derr := hex.DecodeString(sigs[i].Hex)
		if derr != nil {
			fmt.Println("bad sig hex:", sigs[i].Name, derr)
			os.Exit(1)
		}
		sigs[i].Raw = raw
	}

	fmt.Printf("File: %s (%d bytes)\n\n", *in, len(data))

	foundAny := false
	for _, s := range sigs {
		offs := findAll(data, s.Raw)
		if len(offs) == 0 {
			continue
		}
		foundAny = true
		fmt.Printf("[%s] found %d time(s)\n", s.Name, len(offs))
		for _, o := range offs {
			fmt.Printf("  - offset: 0x%X (%d)\n", o, o)
		}
		fmt.Println()
	}

	if !foundAny {
		fmt.Println("No known signatures found. This may be a custom container or encrypted/compressed without standard headers.")
	}
}