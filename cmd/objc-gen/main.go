package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/dennwc/go-apple/generator"
)

var (
	f_xml   = flag.String("xml", "./frameworks/AppKit-xml", "folder with Doxygen XML files")
	f_class = flag.String("class", "NSAlert", "class to generate")
	f_pkg   = flag.String("pkg", "appkit", "Go package name")
)

func main() {
	flag.Parse()

	if err := generate(os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func generate(w io.Writer) error {
	g := generator.NewGenerator()
	if err := g.LoadDoxygen(*f_xml); err != nil {
		return err
	}
	s := g.StructByName(*f_class)
	if s == nil {
		return fmt.Errorf("unknown class: %q", *f_class)
	}
	fmt.Fprintf(w, `package %s

import (
	"github.com/mkrautz/objc"
	"github.com/mkrautz/objc/Foundation"
)

var _ objc.Object
var _ = foundation.NSStringFromString

`, *f_pkg)
	if !s.PrintGoWrapper(w) {
		return fmt.Errorf("failed to generate the class")
	}
	return nil
}
