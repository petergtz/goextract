// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"strings"
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

// This example shows what an AST looks like when printed for debugging.
func main() {
	// src is the input for which we want to print the AST.
	const src = `
package main
import (
"fmt"
)
func main() {
i :=3
println(i)
	fmt.Println(i)
}

func bla(myparam int) int {
	return 0
}
`

	// 3 cases:
	// 1. Pure expression
	// 2. Pure procedural (implies side effects) -> list of statemtents -> no return value
	// 3. Final assignment to local variable -> list of statements where final is an assignment

	// Create the AST by parsing src.
	fset := token.NewFileSet() // positions are relative to fset
	f, err := parser.ParseFile(fset, "", src, 0)
	if err != nil {
		panic(err)
	}

	fmt.Printf(
		"%v:%v %v:%v \n",
		fset.Position(f.Decls[1].Pos()).Line,
		fset.Position(f.Decls[1].Pos()).Column,
		fset.Position(f.Decls[1].End()).Line,
		fset.Position(f.Decls[1].End()).Column,
	)
	visitor := &parentVisitor{fset: fset}
	ast.Walk(visitor, f)
	if posParent == endParent {
		fmt.Println("parent: ")
		ast.Print(fset, posParent)
		fmt.Println("to extract: ")
		for _, node := range nodesToExtract {
			ast.Print(fset, node)
		}

	}

	// assume len(nodes) == 1 ==> simple expression:
	switch posParent.(type) {
	case *ast.AssignStmt:
		for i, rhs := range posParent.(*ast.AssignStmt).Rhs {
			if rhs == nodesToExtract[0] {
				posParent.(*ast.AssignStmt).Rhs[i] =
					&ast.CallExpr{Fun: &ast.Ident{Name: "MyExtractedFunc"}}
			}
		}

	}
	f.Decls = append(f.Decls, &ast.FuncDecl{
		Name: &ast.Ident{Name: "MyExtractedFunc"},
		Type: &ast.FuncType{
			Results: &ast.FieldList{
				List: []*ast.Field{
					&ast.Field{
						Type: &ast.Ident{Name: strings.ToLower(nodesToExtract[0].(*ast.BasicLit).Kind.String())},
					},
				},
			},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ReturnStmt{
					Results: []ast.Expr{
						nodesToExtract[0].(ast.Expr),
					},
				},
			},
		},
	})
	// Print the AST.
	ast.Print(fset, f)
	printer.Fprint(os.Stdout, fset, f)

}
