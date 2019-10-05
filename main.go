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
//   Copyright (C) 2019 framp at linux-tips-and-tricks dot de
//
//   This program is free software: you can redistribute it and/or modify
//   it under the terms of the GNU General Public License as published by
//   the Free Software Foundation, either version 3 of the License, or
//   (at your option) any later version.
//
//   This program is distributed in the hope that it will be useful,
//   but WITHOUT ANY WARRANTY; without even the implied warranty of
//   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//   GNU General Public License for more details.
//
//   You should have received a copy of the GNU General Public License
//   along with this program.  If not, see <http://www.gnu.org/licenses/>.
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
