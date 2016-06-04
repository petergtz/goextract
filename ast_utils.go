package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"

	"github.com/petergtz/goextract/util"
)

func astFrom(filename string) (*token.FileSet, *ast.File) {
	fileSet := token.NewFileSet() // positions are relative to fset
	astFile, err := parser.ParseFile(
		fileSet,
		"",
		util.ReadFileAsStringOrPanic(filename),
		0)
	util.PanicOnError(err)

	createAstFileDump(filename+".ast", fileSet, astFile)

	return fileSet, astFile
}

func createAstFileDump(filename string, fileSet *token.FileSet, astFile *ast.File) {
	file, err := os.Create(filename)
	util.PanicOnError(err)
	defer file.Close()
	ast.Fprint(file, fileSet, astFile, ast.NotNilFilter)
}
