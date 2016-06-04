package main

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"

	"github.com/petergtz/goextract/util"
)

func astFromFile(filename string) (*token.FileSet, *ast.File) {
	return astFromInput(util.ReadFileAsStringOrPanic(filename))
}

func astFromInput(input string) (*token.FileSet, *ast.File) {
	fileSet := token.NewFileSet() // positions are relative to fset
	astFile, err := parser.ParseFile(
		fileSet,
		"",
		input,
		0)
	util.PanicOnError(err)

	return fileSet, astFile
}

func createAstFileDump(filename string, fileSet *token.FileSet, astFile *ast.File) {
	file, err := os.Create(filename)
	util.PanicOnError(err)
	defer file.Close()
	ast.Fprint(file, fileSet, astFile, ast.NotNilFilter)
}

func stringFrom(fileSet *token.FileSet, astFile *ast.File) string {
	buf := new(bytes.Buffer)
	err := printer.Fprint(buf, fileSet, astFile)
	util.PanicOnError(err)
	return buf.String()
}
