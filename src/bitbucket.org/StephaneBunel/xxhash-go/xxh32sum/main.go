//   Copyright (C) 2013, Stéphane Bunel
//   BSD 2-Clause License (http://www.opensource.org/licenses/bsd-license.php)

package main

import (
	xxh "bitbucket.org/StephaneBunel/xxhash-go"
	"flag"
	"fmt"
	"io"
	"os"
)

const (
	VERSION = "0.3"
)

var (
	ShowVersion    bool
	ReadBufferSize int
)

func IOReader(input *os.File) (uint32, error) {
	var (
		err    error
		size   int
		buffer = make([]byte, ReadBufferSize)
		hash   = xxh.New32()
	)

	for {
		size, err = input.Read(buffer)
		if err != nil {
			break
		}
		hash.Write(buffer[0:size])
	}

	if err == io.EOF {
		err = nil
	}

	return hash.Sum32(), err
}

func ShowUsage() {
	fmt.Fprintf(os.Stderr, "Usage: xxh32sum [<OPTIONS>] <filename> [<filename>] [...]\n")
	fmt.Fprintf(os.Stderr, "OPTIONS:\n")
	flag.PrintDefaults()
}

func main() {

	flag.BoolVar(&ShowVersion, "version", false, "Show version")
	flag.IntVar(&ReadBufferSize, "readsize", 1<<20, "Read buffer size")
	flag.Usage = ShowUsage
	flag.Parse()

	if ShowVersion == true {
		fmt.Printf("xxh32sum %s\nCopyright (C) 2013 Stéphane Bunel\nhttps://bitbucket.org/StephaneBunel/xxhash-go\n", VERSION)
		return
	}

	if flag.NArg() == 0 {
		ShowUsage()
		return
	}

	if flag.Args()[0] == "-" {
		h32, err := IOReader(os.Stdin)
		if err != nil {
			fmt.Printf("!<stdin>: %v\n", err)
			return
		}
		fmt.Printf("%08x\n", h32)
		return
	}

	for _, filename := range flag.Args() {
		file, err := os.Open(filename)
		if err != nil {
			fmt.Errorf("%v", err)
			continue
		}
		defer file.Close()

		h32, err := IOReader(file)
		if err != nil {
			fmt.Printf("!%s: %v\n", filename, err)
			continue
		}
		fmt.Printf("%08x\t%s\n", h32, filename)
	}
}
