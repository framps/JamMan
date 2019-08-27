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

// See http://www.piclist.com/techref/io/serial/midi/wave.html

type WAVHeader struct {
	Riff   [4]byte // "RIFF"
	Length int32
	Wave   [4]byte // "WAVE"
}

type FMTHeader struct {
	ID                [4]byte // "FMT "
	EchunkSize        int32
	EwFormatTag       int16
	EwChannels        uint16
	EdwSamplesPerSec  uint32
	EdwAvgBytesPerSec uint32
	EwBlockAlign      uint16
	EwBitsPerSample   uint16
}

func (e FMTHeader) String() string {
	return fmt.Sprintf("%s - L:%d Tag:%d - Channels:%d Samplerate: %d BytesPerSecond:%d FrameSize:%d BitsPerSample:%d",
		string(e.ID[:]), e.EchunkSize, e.EwFormatTag, e.EwChannels, e.EdwSamplesPerSec, e.EdwAvgBytesPerSec, e.EwBlockAlign, e.EwBitsPerSample)
}

type DataChunk struct {
	EchunkID   [4]byte // "data"
	EchunkSize int32
	//waveformData []byte
}

func (e DataChunk) String() string {
	return fmt.Sprintf("%s - Size:%d", string(e.EchunkID[:]), e.EchunkSize)
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
	var skipped, valid int

	for i, h := range hits {
		header := WAVHeader{}
		offset := h[0]
		b := bytes.NewBuffer(dat[offset : offset+12])
		err = binary.Read(b, binary.LittleEndian, &header)
		if err != nil {
			fmt.Printf("Read error %s\n", err.Error())
			os.Exit(42)
		}

		var fmtHeader FMTHeader
		b = bytes.NewBuffer(dat[offset+12 : offset+12+24])
		err = binary.Read(b, binary.LittleEndian, &fmtHeader)
		if err != nil {
			fmt.Printf("Read error %s\n", err.Error())
			os.Exit(42)
		}

		var datHeader DataChunk
		b = bytes.NewBuffer(dat[offset+12+24 : offset+12+24+8])
		err = binary.Read(b, binary.LittleEndian, &datHeader)
		if err != nil {
			fmt.Printf("Read error %s\n", err.Error())
			os.Exit(42)
		}

		s := header.Length + 8

		fn := fmt.Sprintf("Loop%02d.wav", i)
		pre := "Creating "
		if (fmtHeader.EchunkSize == 16 || fmtHeader.EchunkSize == 18) &&
			fmtHeader.EwBlockAlign > 2 && // skip sample metronom wavs
			datHeader.EchunkSize != 0 && // skip empty data sections
			(int32)(h[0])+s < (int32)(len(dat)) {

			bw := bytes.NewBuffer(dat[h[0] : (int32)(h[0])+s])
			err := ioutil.WriteFile(fn, bw.Bytes(), 0644)
			if err != nil {
				fmt.Printf("Write error %s\n", err.Error())
				os.Exit(42)
			}
			fmt.Printf("%sFile: %s - Size: %d - Offset %08x\n", pre, fn, s, h[0])
			fmt.Printf("%s\n", fmtHeader)
			fmt.Printf("%s\n", datHeader)
			valid++
		} else {
			pre = "Skipped "
			skipped++
		}
	}
	fmt.Printf("Hits: %d\n", len(hits))
	fmt.Printf("Valid: %d\n", valid)
	fmt.Printf("Skipped: %d\n", skipped)
}
