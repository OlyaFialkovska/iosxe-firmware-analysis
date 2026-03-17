package main

import (
	"fmt"
	"os"

	"fwparse/internal/app"
)

func main() {
	err := app.Run()
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
