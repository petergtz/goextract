package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"

	"github.com/davecgh/go-spew/spew"
)

func main() {
	fileSet := token.NewFileSet()
	astFile, err := parser.ParseFile(fileSet, os.Args[1], nil, parser.ParseComments)
	if err != nil {
		panic(err)
	}
	spew.Config.ContinueOnMethod = true
	spew.Dump(fileSet)
	fmt.Println()
	fmt.Println()
	fmt.Println()
	spew.Dump(astFile)
}
