package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"unsafe"
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
	Length int32   // filesize - 8 in bytes
	Wave   [4]byte // "WAVE"
}

func (e WAVHeader) String() string {
	return fmt.Sprintf("WAV: %s - L:%d - %s", e.Riff, e.Length+8, e.Wave)
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
	return fmt.Sprintf("FMT: %s - L:%d Tag:%d - Channels:%d Samplerate: %d BytesPerSecond:%d FrameSize:%d BitsPerSample:%d",
		string(e.ID[:]), e.EchunkSize, e.EwFormatTag, e.EwChannels, e.EdwSamplesPerSec, e.EdwAvgBytesPerSec, e.EwBlockAlign, e.EwBitsPerSample)
}

type DataChunk struct {
	EchunkID   [4]byte // "data"
	EchunkSize int32
	//waveformData []byte
}

type Wave struct {
	Wh WAVHeader
	Fh FMTHeader
	Dc DataChunk
}

func (e DataChunk) String() string {
	return fmt.Sprintf("CHUNK: %s - Size:%d", string(e.EchunkID[:]), e.EchunkSize)
}

func HandleError(err error) {
	if err != nil {
		fmt.Printf("Open error %s\n", err.Error())
		os.Exit(42)
	}
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
	hits := re.FindAllIndex(dat, -1)
	var skipped, valid int

	rf, err := os.Open(fileName)
	if err != nil {
		fmt.Printf("Open error %s\n", err.Error())
		os.Exit(42)
	}
	defer rf.Close()

	for i, h := range hits {
		wave := Wave{}
		offset := h[0]
		b := bytes.NewBuffer(dat[offset : offset+int(unsafe.Sizeof(wave))])
		err = binary.Read(b, binary.LittleEndian, &wave)
		if err != nil {
			fmt.Printf("Read error %s\n", err.Error())
			os.Exit(42)
		}

		s := wave.Wh.Length + 8

		fn := fmt.Sprintf("Loop%02d.wav", i)
		pre := "Creating "
		if (wave.Fh.EchunkSize == 16 || wave.Fh.EchunkSize == 18) &&
			wave.Fh.EwBlockAlign > 2 && // skip sample metronom wavs
			wave.Dc.EchunkSize != 0 && // skip empty data sections
			(int32)(h[0])+s < (int32)(len(dat)) {

			fmt.Printf("%s %s\n", pre, wave.Wh)
			fmt.Printf("%s %s\n", pre, wave.Fh)
			fmt.Printf("%s %s\n", pre, wave.Dc)

			wf, err := os.Create(fn)
			HandleError(err)
			defer wf.Close()

			n, err := rf.Seek((int64)(h[0]), 0)
			HandleError(err)
			fmt.Printf("Seeked to %08x\n", n)
			b := make([]byte, s)
			_, err = rf.Read(b)
			HandleError(err)

			_, err = wf.Write(b)
			HandleError(err)

			fmt.Printf("%sFile: %s - Size: %d - Offset %08x\n", pre, fn, s, h[0])
			valid++
		} else {
			pre = "Skipped "
			skipped++
			fmt.Printf("%s %s\n", pre, wave.Wh)
			fmt.Printf("%s %s\n", pre, wave.Fh)
			fmt.Printf("%s %s\n", pre, wave.Dc)
		}
	}
	fmt.Printf("Hits: %d\n", len(hits))
	fmt.Printf("Valid: %d\n", valid)
	fmt.Printf("Skipped: %d\n", skipped)
}
