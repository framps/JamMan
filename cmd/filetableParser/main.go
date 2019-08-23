package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

//######################################################################################################################
//
//    Extract lost JamMan Stereo WAV files from NAND dump
//
//    Copyright (C) 2019 framp at linux-tips-and-tricks dot de
//
//#######################################################################################################################

// C struct -> https://github.com/ubyyj/qnx660/blob/bac16ebb4f22ee2ed53f9a058ae68902333e9713/target/qnx6/usr/include/fs/etfs.h

const ETFS_FNAME_SHORT_LEN = 32

type Etfs_ftable_file struct {
	Efid  int16 /* File id of extra info attached to this file */
	Pfid  int16 /* File id of parent to this file */
	Mode  int32 /* File mode */
	Uid   int32 /* User ID of owner */
	Gid   int32 /* Group ID of owner  */
	Atime int32 /* Time of last access */
	Mtime int32 /* Time of last modification */
	Ctime int32 /* Time of last change */
	Size  int32 /* File size (always 0 for directories) */
	Name  [ETFS_FNAME_SHORT_LEN]byte
}

func (e Etfs_ftable_file) String() string {
	name := string(e.Name[:])
	zero := bytes.Index(e.Name[:], []byte{0})
	var filename string
	if zero >= 0 {
		filename = name[0:zero]
	}
	return fmt.Sprintf("efid:%d - pfid:%d - amctime: %d %d %d - %s", e.Efid, e.Pfid, e.Atime, e.Mtime, e.Ctime, filename)
}

func main() {

	if len(os.Args) != 2 {
		fmt.Printf("Missing filetable file")
		os.Exit(42)
	}

	fileName, _ := filepath.Abs(os.Args[1])
	fmt.Printf("Processing filetable %s\n", fileName)

	dat, err := ioutil.ReadFile(fileName)
	if err != nil {
		fmt.Printf("Read error %s\n", err.Error())
		os.Exit(42)
	}

	fmt.Printf("Filesize: %d\n", len(dat))

	var offset int
	var entry Etfs_ftable_file
	const size = 64
	var cnt int
	for ok := true; ok; ok = offset+size < len(dat) {
		b := bytes.NewBuffer(dat[offset : offset+size])
		err = binary.Read(b, binary.LittleEndian, &entry)
		if err != nil {
			fmt.Printf("Read error %s\n", err.Error())
			os.Exit(42)
		}
		fmt.Printf("Cnt: %d - Offset: %d - %+v\n", cnt, offset, entry)
		offset += size
		cnt++
	}
	fmt.Printf("Entries found: %d\n", cnt)
}