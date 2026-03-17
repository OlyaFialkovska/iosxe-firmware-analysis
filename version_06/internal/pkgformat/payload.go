package pkgformat

import (
	"bytes"
	"fmt"
	"io"
)

func LocatePkgPayloadOffset(reader io.ReaderAt, fileSize int64, pkgOffset int64) (int64, error) {
	readSize := 65536

	remaining := fileSize - pkgOffset
	if remaining <= 0 {
		return 0, fmt.Errorf("pkg offset is outside file")
	}

	if int64(readSize) > remaining {
		readSize = int(remaining)
	}

	data := make([]byte, readSize)
	n, err := reader.ReadAt(data, pkgOffset)
	if err != nil && err != io.EOF {
		return 0, err
	}

	data = data[:n]

	start, _, fields := FindBestPkgFieldStart(data)
	if start != -1 && len(fields) > 0 {
		last := fields[len(fields)-1]
		payloadOffset := pkgOffset + int64(last.Offset+3+last.Length)
		return payloadOffset, nil
	}

	end := bytes.Index(data, []byte("CW_END"))
	if end != -1 {
		payloadOffset := pkgOffset + int64(end+len("CW_END"))
		return payloadOffset, nil
	}

	return 0, fmt.Errorf("could not determine pkg payload offset")
}
