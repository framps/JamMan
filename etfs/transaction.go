package etfs

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"sort"
	"strconv"

	"github.com/framps/JamMan/tools"
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

type Etfs_data [DATA_SIZE]byte

type Etfs_cluster struct {
	Data  [DATA_SIZE]byte
	Trans Etfs_trans
}

type Cluster struct {
	Offset uint32
	Trans  Etfs_trans
}

type Transtable []Cluster

func (s Transtable) Len() int {
	return len(s)
}

func (s Transtable) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s Transtable) Less(i, j int) bool {
	return s[i].Trans.Sequence < s[j].Trans.Sequence
}

type Etfs_transaction_file struct {
	//Data  map[int]Etfs_data
	Data  map[int][DATA_SIZE]byte
	Trans Etfs_trans
}

func NewEtfs_transaction_file() Etfs_transaction_file {
	var tf Etfs_transaction_file
	tf.Data = make(map[int][DATA_SIZE]byte)
	return tf
}

func (e Etfs_transaction_file) String() string {
	return fmt.Sprintf("%s", e.Trans)
}

type Etfs_transaction_file_table []Etfs_transaction_file

// ParseFiletable -
func ParseTransactions(fileName string) (Transtable, error) {

	fmt.Printf("Parsing transactions %s\n", fileName)

	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}

	transTable := make(Transtable, 0, 50000)

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

		var cluster Etfs_cluster
		b := bytes.NewBuffer(bc)
		err = binary.Read(b, binary.LittleEndian, &cluster)
		if err != nil {
			return nil, err
		}

		if cluster.Trans.Fid != UNUSED_FID {
			transCluster := Cluster{offset, cluster.Trans}
			//fmt.Printf("# %06d - Offset: %08x - Trans: %s\n", cnt, offset, cluster.Trans)
			transTable = append(transTable, transCluster)
		}
		cnt++
		offset += CLUSTER_SIZE
	}

	fmt.Printf("Transactions found: %d\n", cnt)

	fmt.Printf("Sorting transactions...\n")
	sort.Sort(transTable)

	return transTable, nil
}

func ProcessTransactions(fileName string, transTable Transtable) (Etfs_transaction_file_table, error) {

	fmt.Printf("Processing transactions %s\n", fileName)

	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	bc := make([]byte, DATA_SIZE+TRANS_SIZE)
	var c Etfs_cluster

	transactionFile := NewEtfs_transaction_file()

	for i := range transTable {
		//fmt.Printf("Offset: %08x Trans: %s\n", transTable[i].Offset, transTable[i].Trans)

		if transTable[i].Trans.Fid == 173 {
			fmt.Printf("READ Offset: %08x Trans: %s\n", transTable[i].Offset, transTable[i].Trans)
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

			b := bytes.NewBuffer(bc)
			err = binary.Read(b, binary.LittleEndian, &c)
			if err != nil {
				return nil, err
			}
			fmt.Printf("Updating cluster %d\n", c.Trans.Cluster)
			transactionFile.Data[(int)(c.Trans.Cluster)] = c.Data
		}
	}

	//fmt.Println(transactionFile.Data)

	wf, err := os.Create("1.wav")
	tools.HandleError(err)
	defer wf.Close()

	keys := make([]string, 0, len(transactionFile.Data))
	for k := range transactionFile.Data {
		keys = append(keys, fmt.Sprintf("%d", k))
	}
	sort.Strings(keys)

	for _, k := range keys {
		i, _ := strconv.Atoi(k)
		//fmt.Println(k, transactionFile.Data[i])
		b := transactionFile.Data[i]
		wf.Write(b[:])
	}

	return Etfs_transaction_file_table{transactionFile}, nil
}
