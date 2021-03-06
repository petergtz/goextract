package main

import (
	"fmt"

	"github.com/petergtz/goextract/util"

	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	inputFilename  = kingpin.Arg("input", "Input filename").Required().String()
	selection      = kingpin.Flag("selection", "begin_line:begin_column-end_line:end_column").Short('s').Required().String()
	funcName       = kingpin.Flag("function", "Name of extracted function").Short('f').Required().String()
	outputFilename = kingpin.Flag("output", "Output filename").Short('o').String()
)

func main() {
	kingpin.Parse()
	adjustedSelection := ShrinkToNonWhiteSpace(selectionFromString(*selection), util.ReadFileAsStringOrPanic(*inputFilename))
	if *outputFilename == "" {
		fmt.Println(ExtractFileToString(*inputFilename, adjustedSelection, *funcName, false))
	} else {
		ExtractFileToFile(*inputFilename, adjustedSelection, *funcName, *outputFilename, false)
	}
}
