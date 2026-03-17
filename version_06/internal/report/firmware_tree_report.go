package report

import (
	"fmt"
	"os"

	"fwparse/internal/model"
)

func SaveFirmwareTreeReport(
	outPath string,
	root *model.Node,
	writeFunc func(*os.File, *model.Node, int),
) error {

	file, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer file.Close()

	fmt.Fprintln(file, "Firmware Structure")
	fmt.Fprintln(file, "---")

	writeFunc(file, root, 0)

	return nil
}
