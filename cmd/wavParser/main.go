package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
)

//######################################################################################################################
//
//    Extract lost JamMan Stereo WAV files from NAND dump
//
//    Copyright (C) 2019 framp at linux-tips-and-tricks dot de
//
//#######################################################################################################################

type WAVHeader struct {
	Riff   [4]byte
	Length int32
	Wav    [4]byte
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
	re := regexp.MustCompile(`RIFF....WAVE`)
	hits := re.FindAllSubmatchIndex(dat, -1)
	fmt.Printf("Hits: %d\n%v\n", len(hits), hits)

	for i, h := range hits {
		header := WAVHeader{}
		offset := h[0]
		b := bytes.NewBuffer(dat[offset : offset+12])
		err = binary.Read(b, binary.LittleEndian, &header)
		if err != nil {
			fmt.Printf("Read error %s\n", err.Error())
			os.Exit(42)
		}

		s := header.Length + 8

		fn := fmt.Sprintf("Loop%02d.wav", i)
		pre := "Creating "
		if s > 1024*1024 &&
			(int32)(h[0])+s < (int32)(len(dat)) {

			bw := bytes.NewBuffer(dat[h[0] : (int32)(h[0])+s])
			err := ioutil.WriteFile(fn, bw.Bytes(), 0644)
			if err != nil {
				fmt.Printf("Write error %s\n", err.Error())
				os.Exit(42)
			}
		} else {
			pre = "Skipped "
		}
		fmt.Printf("%sFile: %s - Size: %d - Offset %x:%x\n", pre, fn, s, h[0], h[1])
	}
}
