package etfs

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"sort"
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

type Etfs_cluster struct {
	Offset uint32
	Trans  Etfs_trans
}

type Etfs_transtable []Etfs_cluster

func (s Etfs_transtable) Len() int {
	return len(s)
}

func (s Etfs_transtable) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s Etfs_transtable) Less(i, j int) bool {
	return s[i].Trans.Sequence < s[j].Trans.Sequence
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

func HandleError(err error) {
	if err != nil {
		fmt.Printf("error %s\n", err.Error())
		os.Exit(42)
	}
}

// ParseFiletable -
func ParseTransactions(fileName string) (Etfs_transtable, error) {

	fmt.Printf("Processing transactions %s\n", fileName)

	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}

	var transTable Etfs_transtable
	transTable = make(Etfs_transtable, 0, 50000)

	bc := make([]byte, DATA_SIZE+TRANS_SIZE)

	var cnt int
	var offset uint32

readLoop:
	for {
		l, err := f.Read(bc)
		if l == 0 {
			break readLoop
		}
		if err != nil {
			return nil, err
		}

		var cluster Etfs
		b := bytes.NewBuffer(bc)
		err = binary.Read(b, binary.LittleEndian, &cluster)
		if err != nil {
			return nil, err
		}

		if cluster.Trans.Fid != UNUSED_FID {
			transCluster := Etfs_cluster{offset, cluster.Trans}
			fmt.Printf("# %06d - Offset: %08x - Trans: %s\n", cnt, offset, cluster.Trans)
			transTable = append(transTable, transCluster)
		}
		cnt++
		offset += CLUSTER_SIZE
	}

	fmt.Printf("Transactions found: %d\n", cnt)

	d := make([]Etfs_data, 0, 10*2048)

	fmt.Printf("Sorting transactions...\n")
	sort.Sort(transTable)

	for i := range transTable {
		fmt.Printf("Offset: %08x Trans: %s\n", transTable[i].Offset, transTable[i].Trans)

		if transTable[i].Trans.Fid == 1 && transTable[i].Trans.Sequence == 0 {

			fmt.Printf("Offset: %08x Trans: %s\n", transTable[i].Offset, transTable[i].Trans)
			l, err := f.Seek((int64)(transTable[i].Offset), 0)
			if err != nil {
				return nil, err
			}
			if l != (int64)(transTable[i].Offset) {
				return nil, fmt.Errorf("Invalid offset. Expected %08x and git %08x", transTable[i].Offset, l)
			}

			_, err = f.Read(bc)
			if err != nil {
				return nil, err
			}

			var cluster Etfs
			b := bytes.NewBuffer(bc)
			err = binary.Read(b, binary.LittleEndian, &cluster)
			if err != nil {
				return nil, err
			}

			d = append(d, cluster.Data)
		}

	}

	return transTable, nil
}
