package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
)

func main() {
	fileSet := token.NewFileSet()
	astFile, err := parser.ParseFile(fileSet, os.Args[1], nil, 0)
	if err != nil {
		panic(err)
	}
	ast.Print(fileSet, astFile)
}
