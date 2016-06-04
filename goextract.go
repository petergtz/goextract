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

type selection struct {
	begin, end position
}

type position struct {
	line, column int
}

type visitorContext struct {
	fset           *token.FileSet
	posParent      ast.Node
	endParent      ast.Node
	nodesToExtract []ast.Node
	shouldRecord   bool

	selection selection
}

type astNodeVisitor struct {
	parentNode ast.Node
	context    *visitorContext
}

func (visitor *astNodeVisitor) Visit(node ast.Node) (w ast.Visitor) {
	if node != nil {
		// fmt.Print("The node: ")
		// ast.Print(visitor.fset, node)
		// fmt.Print("The pos: ", visitor.fset.Position(node.Pos()).Line, ":", visitor.fset.Position(node.Pos()).Column, "\n")
		if visitor.context.fset.Position(node.Pos()).Line == visitor.context.selection.begin.line &&
			visitor.context.fset.Position(node.Pos()).Column == visitor.context.selection.begin.column {
			visitor.context.posParent = visitor.parentNode
			visitor.context.shouldRecord = true
		}
		if visitor.context.shouldRecord {
			visitor.context.nodesToExtract = append(visitor.context.nodesToExtract, node)
		}
		if visitor.context.fset.Position(node.End()).Line == visitor.context.selection.end.line &&
			visitor.context.fset.Position(node.End()).Column == visitor.context.selection.end.column {
			visitor.context.endParent = visitor.parentNode
			visitor.context.shouldRecord = false
		}
	}
	return &astNodeVisitor{
		parentNode: node,
		context:    visitor.context,
	}
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

	doExtraction(fileSet, astFile, selection{position{7, 5}, position{7, 6}}, "MyExtractedFunc")
	createAstFileDump("single_declaration.go.output"+".ast", fileSet, astFile)

	buf := new(bytes.Buffer)
	printer.Fprint(buf, fileSet, astFile)
	RegisterFailHandler(func(message string, callerSkip ...int) { fmt.Println(message) })
	Expect(buf.String()).To(Equal(util.ReadFileAsStringOrPanic("single_declaration.go.output")))

}

func doExtraction(fileSet *token.FileSet, astFile *ast.File, selection selection, extractedFuncName string) {

	visitor := &astNodeVisitor{parentNode: nil, context: &visitorContext{fset: fileSet, selection: selection}}
	ast.Walk(visitor, astFile)
	if visitor.context.posParent != visitor.context.endParent {
		panic("Selection is not valid")
	}
	// if posParent == endParent {
	// 	fmt.Println("parent: ")
	// 	ast.Print(fileSet, posParent)
	// 	fmt.Println("to extract: ")
	// 	for _, node := range nodesToExtract {
	// 		ast.Print(fileSet, node)
	// 	}
	//
	// }

	// assume len(nodes) == 1 ==> simple expression:
	if len(visitor.context.nodesToExtract) == 1 {
		extractExpression(astFile, visitor.context, extractedFuncName)
	} else {
		panic("Only expression extractions are supported so far.")
	}

}

func extractExpression(astFile *ast.File, context *visitorContext, extractedFuncName string) {
	extractedExpressionNode := context.nodesToExtract[0].(ast.Expr)
	switch context.posParent.(type) {
	case *ast.AssignStmt:
		for i, rhs := range context.posParent.(*ast.AssignStmt).Rhs {
			if rhs == extractedExpressionNode {
				context.posParent.(*ast.AssignStmt).Rhs[i] =
					&ast.CallExpr{Fun: &ast.Ident{Name: extractedFuncName}}
			}
		}
		// Add more cases here, e.g. for CallExpr
	}
	astFile.Decls = append(astFile.Decls, &ast.FuncDecl{
		Name: &ast.Ident{Name: extractedFuncName},
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
