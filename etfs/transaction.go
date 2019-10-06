package etfs

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"sort"

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

func (e Cluster) String() string {
	return fmt.Sprintf("Offset:%08x - Trans: %s", e.Offset, e.Trans)
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

type Transaction_file struct {
	Data  map[int]Etfs_data
	Trans Etfs_trans
}

func NewTransaction_file(trans Etfs_trans) Transaction_file {
	var tf Transaction_file
	tf.Data = make(map[int]Etfs_data)
	tf.Trans = trans
	return tf
}

func (e Transaction_file) String() string {
	return fmt.Sprintf("%s", e.Trans)
}

type Transaction_file_table []Transaction_file

// ParseFiletable - read all transaction sections from etfs dump file and sort them in transaction sequence
func ParseTransactions(fileName string) (Transtable, error) {

	fmt.Printf("--- Parsing transactions %s\n", fileName)

	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}

	transTable := make(Transtable, 0, 50000)

	bc := make([]byte, DATA_SIZE+TRANS_SIZE)

	var cnt int
	var offset uint32
	var validTransactions, invalidTransactions int

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
			validTransactions++
			transCluster := Cluster{offset, cluster.Trans}
			//fmt.Printf("+++ %06d - Offset: %08x - Trans: %s\n", cnt, offset, cluster.Trans)
			transTable = append(transTable, transCluster)
		} else {
			invalidTransactions++
			//fmt.Printf("--- %06d - Offset: %08x - Trans: %s\n", cnt, offset, cluster.Trans)
		}
		cnt++
		offset += CLUSTER_SIZE
	}

	fmt.Printf("Transactions found: %d\n", cnt)
	fmt.Printf("Valid transactions: %d\n", validTransactions)
	fmt.Printf("Invalid transactions: %d\n", invalidTransactions)

	fmt.Printf("--- Sorting transactions...\n")
	sort.Sort(transTable)

	return transTable, nil
}

// ProcessTransactions - read cluster data in transaction sequence for all files and renew file clusters if updated in a newer transaction
func ProcessTransactions(fileName string, transTable Transtable) (map[int]Transaction_file, error) {

	fmt.Printf("--- Processing transactions %s\n", fileName)

	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	bc := make([]byte, DATA_SIZE+TRANS_SIZE)
	var c Etfs_cluster

	transactionFileTable := make(map[int]Transaction_file)

	for i := range transTable {
		//fmt.Printf("Offset: %08x Trans: %s\n", transTable[i].Offset, transTable[i].Trans)

		fid := (int)(transTable[i].Trans.Fid)
		if _, ok := transactionFileTable[fid]; !ok {
			transactionFileTable[fid] = NewTransaction_file(transTable[i].Trans)
		}
		//fmt.Printf("READ Offset: %08x Trans: %s\n", transTable[i].Offset, transTable[i].Trans)
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
		(transactionFileTable[fid]).Data[(int)(c.Trans.Cluster)] = c.Data
	}

	//fmt.Println(transactionFile.Data)

	fmt.Printf("Visited transactions: %d\n", len(transTable))

	return transactionFileTable, nil
}

// ExtractFiles - recover all files from their clusters, filename will be <fid>_<filename>.rcvrd
func ExtractFiles(fileTable []Etfs_ftable_file, transactionFileTable map[int]Transaction_file) error {

	fmt.Printf("--- Extracting files\n")

	var cnt int
	var del int
	for i := range transactionFileTable {

		transactionFile := transactionFileTable[i]
		fid := transactionFile.Trans.Fid
		filename := fileTable[fid].Filename()
		deleted := fileTable[fid].isDeleted()

		st := ""
		if deleted {
			st = "_DEL_"
			del++
		}
		fn := fmt.Sprintf("%d_%s%s", fid, st, filename)
		//fmt.Printf("Recovering %d: %s\n", (int)(fileTable[fid].Size), fn)
		wf, err := os.Create(fn + ".rcvrd")
		tools.HandleError(err)
		defer wf.Close()

		// sort clusters in offset sequence
		keys := make([]int, 0, len(transactionFile.Data))
		for k := range transactionFile.Data {
			keys = append(keys, k)
		}

		sort.Slice(keys, func(i, j int) bool {
			return keys[i] < keys[j]
		})

		// write all clusters now
		for i, k := range keys {
			b := transactionFile.Data[k]
			if i < len(keys) {
				wf.Write(b[:])
			} else { // last cluster should be truncated to create a file of file size
				rest := (int)(fileTable[fid].Size) % DATA_SIZE
				wf.Write(b[:rest])
			}
		}
		cnt++
	}
	fmt.Printf("Detected files: %d\n", cnt)
	fmt.Printf("Extracted files: %d\n", cnt-del)
	fmt.Printf("Recovered deleted files (may be incomplete): %d\n", del)

	return nil
}
