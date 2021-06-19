package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/sourcegraph/lsif-static-doc/staticdoc"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Printf("usage: lsif-static-doc dump.lsif project/root/path outdir/\n\n")
		os.Exit(2)
	}
	argDump := os.Args[1]
	argRoot := os.Args[2]
	argOutdir := os.Args[3]

	dump, err := os.Open(argDump)
	if err != nil {
		log.Fatal(err)
	}

	// TODO(slimsag): expose staticdoc.Options as flags.
	files, err := staticdoc.Generate(context.Background(), dump, argRoot, staticdoc.TestingOptions)
	if err != nil {
		log.Fatal(err)
	}

	// Write the files.
	for filePath, fileContents := range files.ByPath {
		filePath = filepath.Join(argOutdir, filePath)
		_ = os.MkdirAll(filepath.Dir(filePath), 0700)
		err := ioutil.WriteFile(filePath, fileContents, 0700)
		if err != nil {
			log.Fatal(err)
		}
	}
}
