package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
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

const ETFS_FILE_END = -1 // pfid for last dummy entry in filetable

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

var etfs []Etfs_ftable_file

func (e Etfs_ftable_file) String() string {
	atime := time.Unix((int64)(e.Atime), 0).Format(time.RFC3339)
	return fmt.Sprintf("%s # efid:%d - pfid:%d - amctime: %s %d %d - %s", e.Status(), e.Efid, e.Pfid, atime, e.Mtime, e.Ctime, e.Filename())
}

func (e Etfs_ftable_file) Filename() string {
	name := string(e.Name[:])
	zero := bytes.Index(e.Name[:], []byte{0})
	var filename string
	if zero >= 0 {
		filename = name[0:zero]
	}
	return filename
}

func (e Etfs_ftable_file) Status() string {
	deleted := "       "
	if e.Efid == -1 {
		deleted = "DELETED"
	}
	return deleted
}

// ParseFiletable -
func ParseFiletable(fileName string) ([]Etfs_ftable_file, error) {

	dump := strings.HasSuffix(fileName, ".img")

	if !dump {
		fmt.Printf("Processing filetable %s\n", fileName)
	} else {
		fmt.Printf("Processing dump %s\n", fileName)
	}

	dat, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Filesize: %d\n", len(dat))

	var offset int
	var entry Etfs_ftable_file
	etfs = make([]Etfs_ftable_file, 0, 500)
	const size = 64
	var cnt int
	for ok := true; ok; ok = offset+size < len(dat) {
		b := bytes.NewBuffer(dat[offset : offset+size])
		err = binary.Read(b, binary.LittleEndian, &entry)
		if err != nil {
			return nil, err
		}
		if entry.Pfid == ETFS_FILE_END {
			break
		}
		fmt.Printf("Buffer: %x\n", dat[offset:offset+size])
		fmt.Printf("Cnt: %d - Offset: %d - %+v\n", cnt, offset, entry)
		etfs = append(etfs, entry)
		offset += size
		cnt++
	}

	fmt.Printf("Entries found: %d\n", cnt)
	return etfs, nil
}

func main() {

	if len(os.Args) != 2 {
		fmt.Printf("Missing filetable file")
		os.Exit(42)
	}

	fileName, _ := filepath.Abs(os.Args[1])

	etfs, err := ParseFiletable(fileName)
	if err != nil {
		fmt.Printf("Error reading %s: %s\n", fileName, err)
	}

	os.Exit(0)

	for i := range etfs {
		c := etfs[i]
		name := c.Filename()
		for {
			if c.Pfid == 0 {
				break
			}
			c = etfs[c.Pfid]
			name = c.Filename() + "/" + name
		}
		fmt.Printf("%03d: %s %s\n", i, etfs[i].Status(), name)
	}
}
