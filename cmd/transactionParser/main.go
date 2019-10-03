package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
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

const DATA_SIZE = 0x800
const TRANS_SIZE = 16
const CLUSTER_SIZE = DATA_SIZE + TRANS_SIZE

const ETFS_TRANS_OK = 0      /* A valid transaction */
const ETFS_TRANS_ECC = 1     /* A valid transaction which was corrected by the ECC */
const ETFS_TRANS_ERASED = 2  /* An erased block with an erase stamp */
const ETFS_TRANS_FOXES = 3   /* An erased block */
const ETFS_TRANS_DATAERR = 5 /* crc error (ECC could not correct) */
const ETFS_TRANS_DEVERR = 6  /* Device retuned a hardware error */
const ETFS_TRANS_BADBLK = 7  /* Bad blk marked at the factory (NAND flash only) */
const ETFS_TRANS_MASK = 0x0f /* Mask for above status */

const UNUSED_CLUSTER = 0xffffff
const UNUSED_FID = 0xffff

type Etfs_trans struct {
	Fid       uint32 /* File id */
	Cluster   uint32 /* Cluster offset in file */
	Nclusters uint16 /* Number of contiguious clusters for this transaction */
	Tacode    uint8  /* Code for transaction area */
	Dacode    uint8  /* Code for data area */
	Sequence  uint32 /* Sequence for this transaction */
}

func (e Etfs_trans) String() string {
	return fmt.Sprintf("Fid:%08d - Cluster:%08x - NClusters:%08d - Tacode:%d - Dacode:%d - Sequence:%08d", e.Fid, e.Cluster, e.Nclusters, e.Tacode, e.Dacode, e.Sequence)
}

type Etfs_data struct {
	Binary [DATA_SIZE]byte
}

type Etfs struct {
	Data  Etfs_data
	Trans Etfs_trans
}

func (e Etfs) String() string {
	return fmt.Sprintf("%s", e.Trans)
}

// ParseFiletable -
func ParseTransactions(fileName string) error {

	fmt.Printf("Processing transactions %s\n", fileName)

	f, err := os.Open(fileName)
	if err != nil {
		return err
	}

	bc := make([]byte, DATA_SIZE+TRANS_SIZE)

	var cnt int
	var offset int

readLoop:
	for {
		l, err := f.Read(bc)
		if l == 0 {
			break readLoop
		}
		if err != nil {
			return err
		}

		var cluster Etfs
		b := bytes.NewBuffer(bc)
		err = binary.Read(b, binary.LittleEndian, &cluster)
		if err != nil {
			return err
		}

		if cluster.Trans.Fid != UNUSED_FID {
			fmt.Printf("Cnt: %04d - Offset: %08x - Trans: %s\n", cnt, offset, cluster.Trans)
		}
		cnt++
		offset += CLUSTER_SIZE
	}

	fmt.Printf("Clusters: %d\n", cnt)
	return nil
}

func main() {

	if len(os.Args) != 2 {
		fmt.Printf("Missing filetable file")
		os.Exit(42)
	}

	fileName, _ := filepath.Abs(os.Args[1])

	err := ParseTransactions(fileName)
	if err != nil {
		fmt.Printf("Error reading %s: %s\n", fileName, err)
	}

}
