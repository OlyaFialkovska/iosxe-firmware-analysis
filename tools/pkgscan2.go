package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

type Sig struct {
	Name string
	Needle []byte
}

type Hit struct {
	Signature string `json:"signature"`
	OffsetHex string `json:"offset_hex"`
	OffsetDec int    `json:"offset_dec"`
}

type CarvedFile struct {
	FromSig    string `json:"from_signature"`
	OffsetHex  string `json:"offset_hex"`
	OffsetDec  int    `json:"offset_dec"`
	BytesWrote int64  `json:"bytes_written"`
	SHA256     string `json:"sha256"`
	Path       string `json:"path"`
	HeadHex    string `json:"head_hex"`
}

type Report struct {
	InputFile    string       `json:"input_file"`
	FileSize     int64        `json:"file_size"`
	Hits         []Hit        `json:"hits"`
	Carved       []CarvedFile `json:"carved"`
	Notes        []string     `json:"notes"`
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

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil { return "", err }
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func headHex(path string, n int) (string, error) {
	f, err := os.Open(path)
	if err != nil { return "", err }
	defer f.Close()

	buf := make([]byte, n)
	r, _ := f.Read(buf)
	return hex.EncodeToString(buf[:r]), nil
}

func main() {
	in := flag.String("in", "", "path to .pkg file")
	out := flag.String("out", "report.json", "output JSON report path")
	doCarve := flag.Bool("carve", false, "carve chunks starting at found offsets")
	maxCarveMB := flag.Int("maxmb", 128, "max MB to carve per chunk (safety limit)")
	flag.Parse()

	if *in == "" {
		fmt.Println("Usage: pkgscan2 -in <file.pkg> [-carve] [-maxmb 128] [-out report.json]")
		os.Exit(1)
	}

	data, err := os.ReadFile(*in)
	if err != nil {
		fmt.Println("read error:", err)
		os.Exit(1)
	}

	fi, _ := os.Stat(*in)
	rep := Report{
		InputFile: *in,
		FileSize:  fi.Size(),
	}

	// Signatures (byte patterns)
	sigs := []Sig{
		{Name: "ELF", Needle: []byte{0x7f, 'E', 'L', 'F'}},
		{Name: "GZIP", Needle: []byte{0x1f, 0x8b}},
		{Name: "XZ", Needle: []byte{0xfd, 0x37, 0x7a, 0x58, 0x5a, 0x00}},
		{Name: "CPIO_NEWC_ASCII_070701", Needle: []byte("070701")},
		{Name: "SQUASHFS_hsqs", Needle: []byte("hsqs")}, // squashfs magic can appear as hsqs in byte view
		{Name: "ZIP_PK", Needle: []byte("PK\x03\x04")},
		{Name: "TAR_USTAR", Needle: []byte("ustar")},
	}

	// Scan
	var hits []Hit
	for _, s := range sigs {
		offs := findAll(data, s.Needle)
		for _, o := range offs {
			hits = append(hits, Hit{
				Signature: s.Name,
				OffsetHex: fmt.Sprintf("0x%X", o),
				OffsetDec: o,
			})
		}
	}

	// Sort hits by offset
	sort.Slice(hits, func(i, j int) bool { return hits[i].OffsetDec < hits[j].OffsetDec })
	rep.Hits = hits

	if len(hits) == 0 {
		rep.Notes = append(rep.Notes, "No known magic signatures found. Container may be proprietary/encrypted/compressed without standard headers.")
	}

	// Carve chunks
	if *doCarve && len(hits) > 0 {
		outDir := "carved"
		_ = os.MkdirAll(outDir, 0755)

		maxBytes := int64(*maxCarveMB) * 1024 * 1024

		// For carving, we carve from each hit offset to next hit offset (or maxBytes), whichever is smaller.
		for i, h := range hits {
			start := int64(h.OffsetDec)
			var end int64

			if i < len(hits)-1 {
				end = int64(hits[i+1].OffsetDec)
			} else {
				end = int64(len(data))
			}

			// apply safety cap
			if end-start > maxBytes {
				end = start + maxBytes
			}
			if start < 0 || start >= int64(len(data)) || end <= start {
				continue
			}

			chunk := data[start:end]
			name := fmt.Sprintf("%04d_%s_%s.bin", i, h.Signature, h.OffsetHex)
			path := filepath.Join(outDir, name)

			if err := os.WriteFile(path, chunk, 0644); err != nil {
				rep.Notes = append(rep.Notes, fmt.Sprintf("carve failed at %s: %v", h.OffsetHex, err))
				continue
			}

			sum, _ := sha256File(path)
			hh, _ := headHex(path, 32)

			rep.Carved = append(rep.Carved, CarvedFile{
				FromSig:    h.Signature,
				OffsetHex:  h.OffsetHex,
				OffsetDec:  h.OffsetDec,
				BytesWrote: int64(len(chunk)),
				SHA256:     sum,
				Path:       path,
				HeadHex:    hh,
			})
		}

		rep.Notes = append(rep.Notes, "Carving strategy: each chunk starts at a found signature offset and ends at the next signature offset (or maxMB limit).")
		rep.Notes = append(rep.Notes, "Use binwalk/strings/file on carved chunks for deeper iterative analysis.")
	}

	// Write JSON report
	b, _ := json.MarshalIndent(rep, "", "  ")
	if err := os.WriteFile(*out, b, 0644); err != nil {
		fmt.Println("write report error:", err)
		os.Exit(1)
	}

	fmt.Printf("OK: %d hits. Report written to %s\n", len(rep.Hits), *out)
	if *doCarve {
		fmt.Printf("Carved %d chunk(s) into ./carved/\n", len(rep.Carved))
	}
}
