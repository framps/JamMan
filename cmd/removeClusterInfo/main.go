package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"os"
)

//######################################################################################################################
//
//    Extract lost JamMan Stereo WAV files from NAND dump
//
//    Copyright (C) 2019 framp at linux-tips-and-tricks dot de
//
//#######################################################################################################################

// See http://www.piclist.com/techref/io/serial/midi/wave.html

const EMPTY_CLUSTER = 0x00ffffff

type Etfs_cluster_separator struct {
	ClusterNumber uint32
	BlockNumber   uint32
	BlocksLeft    uint32
	Misc          uint32
}

func (e Etfs_cluster_separator) String() string {
	return fmt.Sprintf("%08x - %08x - %08x - %d", e.ClusterNumber, e.BlockNumber, e.BlocksLeft, e.Misc)
}

func main() {

	if len(os.Args) != 2 {
		fmt.Printf("Missing NAND dump file")
		os.Exit(42)
	}

	fileName := os.Args[1]
	fmt.Printf("Processing NAND dump %s\n", fileName)

	dat, err := ioutil.ReadFile(fileName)
	if err != nil {
		fmt.Printf("Read error %s\n", err.Error())
		os.Exit(42)
	}

	fmt.Printf("Size: %d\n", len(dat))

	f, err := os.Create(fileName + ".strp")
	defer f.Close()

	var cluster int

	for i := 0; i < len(dat); i += 0x800 + 16 {

		b := bytes.NewBuffer(dat[i+0x800 : i+0x800+16])
		var filler Etfs_cluster_separator
		err = binary.Read(b, binary.LittleEndian, &filler)

		if filler.BlockNumber != EMPTY_CLUSTER {

			cluster++

			fmt.Printf("Cluster %08d: %s\n", cluster, filler)
			l := 0x800
			if i+l > len(dat) {
				l = len(dat) - i
			}
			_, err := f.Write(dat[i : i+l])
			if err != nil {
				fmt.Printf("Error writing %s: %s\n", fileName+"strp", err)
			} else {
				fmt.Printf("Written cluster %d\n", cluster)
			}
		}
	}
}
