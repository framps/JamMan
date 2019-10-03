package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

const CLUSTER_SIZE = 2048

const EMPTY_CLUSTER = 0x00ffffff

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
	//atime := time.Unix((int64)(e.Atime), 0).Format(time.RFC3339)
	//mtime := time.Unix((int64)(e.Mtime), 0).Format(time.RFC3339)
	//ctime := time.Unix((int64)(e.Ctime), 0).Format(time.RFC3339)
	return fmt.Sprintf("%s # efid:%d - pfid:%d - times: %08x %08x %08x - %s:%d", e.Status(), e.Efid, e.Pfid, e.Atime, e.Mtime, e.Ctime, e.Filename(), e.Size)
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

var etfs []Etfs_ftable_file

type Etfs_cluster_separator struct {
	ClusterNumber uint32
	BlockNumber   uint32
	BlocksLeft    uint32
	Misc          uint32
}

func (e Etfs_cluster_separator) String() string {
	return fmt.Sprintf("ClusterNumber:%08d - Blocknumber:%08d - Blocksleft:%08d - Misc:%d", e.ClusterNumber, e.BlockNumber, e.BlocksLeft, e.Misc)
}

// ParseFiletable -
func ParseFiletable(fileName string) ([]Etfs_ftable_file, error) {

	var validClusters = 0

	dump := strings.HasSuffix(fileName, ".img")

	if !dump {
		fmt.Printf("Processing filetable %s\n", fileName)
	} else {
		fmt.Printf("Processing dump %s\n", fileName)
	}

	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}

	bf := make([]byte, CLUSTER_SIZE)
	bd := make([]byte, 16)

	var globalOffset int
	var entry Etfs_ftable_file
	etfs = make([]Etfs_ftable_file, 0, 500)
	const size = 64
	var cnt int

	var leadingFileTableProsessing = true

readLoop:
	for {
		fmt.Printf("Reading cluster %d...\n", cnt)
		l, err := f.Read(bf)
		if l == 0 {
			break readLoop
		}
		if err != nil {
			return nil, err
		}
		var offset int
		for ok := true; ok; ok = offset+size <= CLUSTER_SIZE {
			// process leading filetable
			if leadingFileTableProsessing {
				b := bytes.NewBuffer(bf[offset : offset+size])
				err = binary.Read(b, binary.LittleEndian, &entry)
				if err != nil {
					return nil, err
				}

				if entry.Pfid == ETFS_FILE_END {
					leadingFileTableProsessing = false
				} else {
					//fmt.Printf("Buffer: %x\n", bf[offset:offset+size])
					fmt.Printf("Cnt: %04d - AbsOffset: %08x - RelOffset: %08x - %s\n", cnt, globalOffset, offset, entry)
					etfs = append(etfs, entry)
					cnt++
				}
			}
			offset += size
			globalOffset += size
		}

		//fmt.Printf("Reading cluster\n")

		if dump {
			l, err = f.Read(bd)
			if err != nil {
				return nil, err
			}
			if l == 0 {
				break readLoop
			}
			b := bytes.NewBuffer(bd)
			var filler Etfs_cluster_separator
			err = binary.Read(b, binary.LittleEndian, &filler)
			if err != nil {
				return nil, err
			}
			if filler.BlockNumber != EMPTY_CLUSTER {
				//fmt.Printf("Clusterfill: %06x - %x\n", globalOffset, bd)
				fmt.Printf("Clusterfill: %08x - %s\n", globalOffset, filler)
				validClusters++
			}
			globalOffset += 16
		}
	}

	if dump {
		fmt.Printf("Valid clusters: %d\n", validClusters)
	} else {
		fmt.Printf("Entries found: %d\n", cnt)
	}
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
