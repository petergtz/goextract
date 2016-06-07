package main

import (
	"fmt"
	"strings"

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
	if *outputFilename == "" {
		fmt.Println(ExtractFileToString(*inputFilename, selectionFromString(*selection), *funcName))
	} else {
		ExtractFileToFile(*inputFilename, selectionFromString(*selection), *funcName, *outputFilename)
	}
}

// TODO: error handling. Do this with regex
func selectionFromString(s string) Selection {
	s = strings.Replace(s, " ", "", -1)
	beginEnd := strings.Split(s, "-")
	beginString := beginEnd[0]
	endString := beginEnd[1]
	begin := strings.Split(beginString, ":")
	end := strings.Split(endString, ":")

	return Selection{
		Begin: Position{util.ToInt(begin[0]), util.ToInt(begin[1])},
		End:   Position{util.ToInt(end[0]), util.ToInt(end[1])},
	}
}
