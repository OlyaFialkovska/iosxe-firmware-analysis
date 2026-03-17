package util

import (
	"fmt"
	"os"
	"strings"

	"fwparse/internal/model"
)

func WriteTree(file *os.File, node *model.Node, level int) {
	prefix := strings.Repeat("  ", level)

	if node.Details != "" {
		fmt.Fprintf(
			file,
			"%s- %s | offset=0x%X | size=%d | type=%s | %s\n",
			prefix,
			node.Name,
			node.Offset,
			node.Size,
			node.Type,
			node.Details,
		)
	} else {
		fmt.Fprintf(
			file,
			"%s- %s | offset=0x%X | size=%d | type=%s\n",
			prefix,
			node.Name,
			node.Offset,
			node.Size,
			node.Type,
		)
	}

	for _, child := range node.Children {
		WriteTree(file, child, level+1)
	}
}
