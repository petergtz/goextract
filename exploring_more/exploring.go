package main

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"

	"fmt"

	"github.com/davecgh/go-spew/spew"
)

func main2() {

	astFile := &ast.File{Name: ast.NewIdent("Bla"), Decls: []ast.Decl{&ast.GenDecl{Tok: token.VAR, Specs: []ast.Spec{}}}}
	fmt.Println("astFile:")
	spew.Dump(astFile)
	fmt.Println()

	buf := new(bytes.Buffer)
	err := printer.Fprint(buf, token.NewFileSet(), astFile)
	if err != nil {
		panic(err)
	}
	fmt.Println(buf)

}

func main() {
	fileSet := token.NewFileSet()
	astFile, err := parser.ParseFile(fileSet, "minimal.go", nil, parser.ParseComments)
	if err != nil {
		panic(err)
	}
	ast.Print(fileSet, astFile)

	return

	astFile.Decls = append(astFile.Decls[:0], append([]ast.Decl{&ast.FuncDecl{
		Name: ast.NewIdent("MyNewFunc"),
		Type: &ast.FuncType{
			Func: 15,
			Params: &ast.FieldList{
				Opening: 29,
				Closing: 30,
			},
		},
	}}, astFile.Decls[0:]...)...)

	offset := astFile.Decls[0].End() - astFile.Decls[0].Pos()
	intoffset := int(offset)
	fmt.Println("offset", offset)
	astFile.Comments[0].List[0].Slash += offset
	// astFile.Comments[0].List[0].Slash = 18
	astFile.Decls[1].(*ast.GenDecl).TokPos += offset
	astFile.Decls[1].(*ast.GenDecl).Lparen += offset
	astFile.Decls[1].(*ast.GenDecl).Rparen += offset
	fileSetFile := fileSet.File(1)
	newFileSet := token.NewFileSet()
	newFileSetFile := newFileSet.AddFile("whatever", 1, fileSetFile.Size()+int(offset))
	newFileSetFile.SetLines([]int{
		0,
		13,
		14,
		15,
		15 + intoffset,
		20 + intoffset,
		21 + intoffset,
		32 + intoffset,
		33 + intoffset,
	}) // hardcoded for now

	fmt.Println("astFile:")
	spew.Dump(astFile)
	fmt.Println()
	fmt.Println()
	fmt.Println("fileSet:")
	spew.Dump(fileSet)
	buf := new(bytes.Buffer)

	err = printer.Fprint(buf, newFileSet, astFile)
	if err != nil {
		panic(err)
	}

	fmt.Println(buf)

}
