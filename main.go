package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/framps/JamMan/etfs"
	"github.com/framps/JamMan/tools"
)

func main() {

	if len(os.Args) != 3 {
		fmt.Printf("Missing .filetable and dump file")
		os.Exit(42)
	}

	fileTableFilename, _ := filepath.Abs(os.Args[1])
	dumpFilename, _ := filepath.Abs(os.Args[2])

	fmt.Printf("Processing %s and %s\n", fileTableFilename, dumpFilename)

	fileTable, err := etfs.ParseFiletable(fileTableFilename)
	tools.HandleError(err)

	for fid, entry := range fileTable {
		fmt.Printf("Fid: %04d - %s\n", fid, entry)
	}
}
