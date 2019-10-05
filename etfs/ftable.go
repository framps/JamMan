package etfs

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io/ioutil"
)

//######################################################################################################################
//
//    Extract JamMan Stereo WAV files from NAND dump
//
//    Copyright (C) 2019 framp at linux-tips-and-tricks dot de
//
//#######################################################################################################################

// etfs overview -> http://qnx.symmetry.com.au/resources/whitepapers/qnx_flash_memory_for_embedded_paper_RIM_MC411.65.pdf
// etfs C struct -> https://github.com/ubyyj/qnx660/blob/bac16ebb4f22ee2ed53f9a058ae68902333e9713/target/qnx6/usr/include/fs/etfs.h
// etfs creation C code -> https://github.com/vocho/openqnx/blob/master/trunk/utils/m/mkxfs/mkxfs/mk_et_fsys.c

const ETFS_FNAME_SHORT_LEN = 32
const FID_DELETED = -1
const FID_END = -1 // pfid for last dummy entry in filetable

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

const ETFS_FTABLE_SIZE = 64

func (e Etfs_ftable_file) String() string {
	//atime := time.Unix((int64)(e.Atime), 0).Format(time.RFC3339)
	//mtime := time.Unix((int64)(e.Mtime), 0).Format(time.RFC3339)
	//ctime := time.Unix((int64)(e.Ctime), 0).Format(time.RFC3339)
	return fmt.Sprintf("%s # efid:%03d - pfid:%03d - Size:%8d - %s", e.Status(), e.Efid, e.Pfid, e.Size, e.Filename())
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
	status := "OK"
	if e.Efid == FID_DELETED {
		status = "DEL"
	}
	return status
}

// ParseFiletable - parse .filetable
func ParseFiletable(fileName string) ([]Etfs_ftable_file, error) {

	fmt.Printf("--- Parsing filetable %s\n", fileName)

	filetableContents, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	var entry Etfs_ftable_file
	filetable := make([]Etfs_ftable_file, 0, 500)
	var defined int
	var deleted int
	var cnt int

	for offset := 0; offset+ETFS_FTABLE_SIZE < len(filetableContents); offset += ETFS_FTABLE_SIZE {
		b := bytes.NewBuffer(filetableContents[offset : offset+ETFS_FTABLE_SIZE])
		err = binary.Read(b, binary.LittleEndian, &entry)
		if err != nil {
			return nil, err
		}

		if entry.Pfid == FID_END {
			break
		}

		fmt.Printf("Fid: %04d - Offset: %08x - %s\n", cnt, offset, entry)
		filetable = append(filetable, entry)
		cnt++
		if entry.Efid == FID_DELETED {
			deleted++
		} else {
			defined++
		}
	}

	fmt.Printf("Filetable entries found: Defined:%d - Deleted:%d\n", defined, deleted)
	return filetable, nil
}
