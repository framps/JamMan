package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/framps/JamMan/etfs"
	"github.com/framps/JamMan/tools"
)

//######################################################################################################################
//
//   Extract JamMan Stereo WAV files from NAND dump
//
//   No elegant and fast code because a lot of trial and error coding happened but finally it recovers all files
//
//   Copyright (c) 2019 framp at linux-tips-and-tricks dot de
//
//   Permission is hereby granted, free of charge, to any person obtaining a copy
//   of this software and associated documentation files (the "Software"), to deal
//   in the Software without restriction, including without limitation the rights
//   to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
//   copies of the Software, and to permit persons to whom the Software is
//   furnished to do so, subject to the following conditions:
//
//   The above copyright notice and this permission notice shall be included in all
//   copies or substantial portions of the Software.
//
//   THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
//   IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
//   FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
//   AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
//   LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
//   OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
//   SOFTWARE.
//
//#######################################################################################################################

// etfs overview -> http://qnx.symmetry.com.au/resources/whitepapers/qnx_flash_memory_for_embedded_paper_RIM_MC411.65.pdf
// etfs C struct -> https://github.com/ubyyj/qnx660/blob/bac16ebb4f22ee2ed53f9a058ae68902333e9713/target/qnx6/usr/include/fs/etfs.h
// etfs creation C code -> https://github.com/vocho/openqnx/blob/master/trunk/utils/m/mkxfs/mkxfs/mk_et_fsys.c

func main() {

	if len(os.Args) != 3 {
		fmt.Printf("Missing .filetable and/or etfs dump file")
		os.Exit(42)
	}

	fileTableFilename, _ := filepath.Abs(os.Args[1])
	transactionFilename, _ := filepath.Abs(os.Args[2])

	fmt.Printf("--- Processing %s and %s\n", fileTableFilename, transactionFilename)

	fileTable, err := etfs.ParseFiletable(fileTableFilename)
	tools.HandleError(err)

	transactionTable, err := etfs.ParseTransactions(transactionFilename)
	tools.HandleError(err)

	etfs_transactionFiletable, err := etfs.ProcessTransactions(transactionFilename, transactionTable)
	tools.HandleError(err)

	etfs.ExtractFiles(fileTable, etfs_transactionFiletable)

}
