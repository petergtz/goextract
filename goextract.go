// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"strings"

	. "github.com/onsi/gomega"
	"github.com/petergtz/goextract/util"
)

type parentVisitor struct {
	parent       ast.Node
	fset         *token.FileSet
	shouldRecord bool
}

var (
	posParent      ast.Node
	endParent      ast.Node
	nodesToExtract = make([]ast.Node, 0)
)

func (visitor *parentVisitor) Visit(node ast.Node) (w ast.Visitor) {
	if node != nil {
		// fmt.Print("The node: ")
		// ast.Print(visitor.fset, node)
		// fmt.Print("The pos: ", visitor.fset.Position(node.Pos()).Line, ":", visitor.fset.Position(node.Pos()).Column, "\n")
		if visitor.fset.Position(node.Pos()).Line == 7 &&
			visitor.fset.Position(node.Pos()).Column == 5 {
			posParent = visitor.parent
			visitor.shouldRecord = true
		}
		if visitor.shouldRecord {
			nodesToExtract = append(nodesToExtract, node)
		}
		if visitor.fset.Position(node.End()).Line == 7 &&
			visitor.fset.Position(node.End()).Column == 6 {
			endParent = visitor.parent
			visitor.shouldRecord = false
		}
	}
	return &parentVisitor{parent: node, fset: visitor.fset}
}

func main() {

	// 3 cases:
	// 1. Pure expression
	// 2. Pure procedural (implies side effects) -> list of statemtents -> no return value
	// 3. Final assignment to local variable -> list of statements where final is an assignment

	fileSet, astFile := astFrom("single_declaration.go.input")

	// fmt.Println(
	// 	fileSet.Position(astFile.Decls[1].Pos()),
	// 	fileSet.Position(astFile.Decls[1].End()),
	// )

	doExtraction(fileSet, astFile)
	createAstFileDump("single_declaration.go.output"+".ast", fileSet, astFile)

	buf := new(bytes.Buffer)
	printer.Fprint(buf, fileSet, astFile)
	RegisterFailHandler(func(message string, callerSkip ...int) { fmt.Println(message) })
	Expect(buf.String()).To(Equal(util.ReadFileAsStringOrPanic("single_declaration.go.output")))

}

func doExtraction(fileSet *token.FileSet, astFile *ast.File) {

	visitor := &parentVisitor{fset: fileSet}
	ast.Walk(visitor, astFile)
	if posParent == endParent {
		fmt.Println("parent: ")
		ast.Print(fileSet, posParent)
		fmt.Println("to extract: ")
		for _, node := range nodesToExtract {
			ast.Print(fileSet, node)
		}

	}

	// assume len(nodes) == 1 ==> simple expression:

	extractedExpressionNode := nodesToExtract[0].(ast.Expr)
	switch posParent.(type) {
	case *ast.AssignStmt:
		for i, rhs := range posParent.(*ast.AssignStmt).Rhs {
			if rhs == extractedExpressionNode {
				posParent.(*ast.AssignStmt).Rhs[i] =
					&ast.CallExpr{Fun: &ast.Ident{Name: "MyExtractedFunc"}}
			}
		}
		// Add more cases here, e.g. for CallExpr
	}
	astFile.Decls = append(astFile.Decls, &ast.FuncDecl{
		Name: &ast.Ident{Name: "MyExtractedFunc"},
		Type: &ast.FuncType{
			Results: &ast.FieldList{
				List: []*ast.Field{
					&ast.Field{
						Type: &ast.Ident{Name: strings.ToLower(extractedExpressionNode.(*ast.BasicLit).Kind.String())},
					},
				},
			},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ReturnStmt{
					Results: []ast.Expr{
						extractedExpressionNode,
					},
				},
			},
		},
	})

}
