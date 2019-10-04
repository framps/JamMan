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
	transactionFilename, _ := filepath.Abs(os.Args[2])

	fmt.Printf("Processing %s and %s\n", fileTableFilename, transactionFilename)

	fileTable, err := etfs.ParseFiletable(fileTableFilename)
	tools.HandleError(err)

	for fid, entry := range fileTable {
		fmt.Printf("Fid: %04d - %s\n", fid, entry)
	}

	transactionTable, err := etfs.ParseTransactions(transactionFilename)

	for i, entry := range transactionTable {
		fmt.Printf("#: %08d - %s\n", i, entry)
	}
}
