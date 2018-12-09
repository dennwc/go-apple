package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/dennwc/go-apple/generator"
)

var (
	f_xml = flag.String("xml", "./frameworks/AppKit-xml", "folder with Doxygen XML files")
	f_dir = flag.String("dir", "./gen", "directory to write files to")
	f_pkg = flag.String("pkg", "appkit", "Go package name")
)

func main() {
	flag.Parse()

	if err := generate(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func generate() error {
	g, err := generator.NewGenerator(*f_pkg, *f_dir)
	if err != nil {
		return err
	}
	if err := g.LoadDoxygen(*f_xml); err != nil {
		return err
	}
	return g.GenerateAll()
}
