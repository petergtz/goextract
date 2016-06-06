// Copyright 2015 Peter Goetz
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"reflect"
	"strings"

	"github.com/petergtz/goextract/util"
)

type Selection struct {
	Begin, End Position
}

type Position struct {
	Line, Column int
}

type visitorContext struct {
	fset           *token.FileSet
	posParent      ast.Node
	endParent      ast.Node
	nodesToExtract []ast.Node
	shouldRecord   bool

	selection Selection
}

type astNodeVisitor struct {
	parentNode ast.Node
	context    *visitorContext
}

func (visitor *astNodeVisitor) Visit(node ast.Node) (w ast.Visitor) {
	if node != nil {
		if visitor.context.fset.Position(node.Pos()).Line == visitor.context.selection.Begin.Line &&
			visitor.context.fset.Position(node.Pos()).Column == visitor.context.selection.Begin.Column {
			fmt.Print("The node: ")
			ast.Print(visitor.context.fset, node)
			fmt.Print("The pos: ", visitor.context.fset.Position(node.Pos()), "\n")
			visitor.context.posParent = visitor.parentNode
			visitor.context.shouldRecord = true
		}
		if visitor.context.shouldRecord {
			visitor.context.nodesToExtract = append(visitor.context.nodesToExtract, node)
		}
		if visitor.context.fset.Position(node.End()).Line == visitor.context.selection.End.Line &&
			visitor.context.fset.Position(node.End()).Column == visitor.context.selection.End.Column {
			fmt.Print("The node: ")
			ast.Print(visitor.context.fset, node)
			fmt.Print("The end: ", visitor.context.fset.Position(node.End()), "\n")
			visitor.context.endParent = visitor.parentNode
			visitor.context.shouldRecord = false
			return nil
		}
	}
	return &astNodeVisitor{
		parentNode: node,
		context:    visitor.context,
	}
}

// 3 cases:
// 1. Pure expression
// 2. Pure procedural (implies side effects) -> list of statemtents -> no return value
// 3. Final assignment to local variable -> list of statements where final is an assignment

// fmt.Println(
// 	fileSet.Position(astFile.Decls[1].Pos()),
// 	fileSet.Position(astFile.Decls[1].End()),
// )

func ExtractFileToFile(inputFileName string, selection Selection, extractedFuncName string, outputFilename string) {
	fileSet, astFile := astFromFile(inputFileName)
	createAstFileDump(inputFileName+".ast", fileSet, astFile)
	doExtraction(fileSet, astFile, selection, extractedFuncName)
	util.WriteFileAsStringOrPanic(outputFilename, stringFrom(fileSet, astFile))
}

func ExtractFileToString(inputFileName string, selection Selection, extractedFuncName string) string {
	fileSet, astFile := astFromFile(inputFileName)
	createAstFileDump(inputFileName+".ast", fileSet, astFile)
	doExtraction(fileSet, astFile, selection, extractedFuncName)
	return stringFrom(fileSet, astFile)
}

func ExtractStringToString(input string, selection Selection, extractedFuncName string) string {
	fileSet, astFile := astFromInput(input)
	doExtraction(fileSet, astFile, selection, extractedFuncName)
	return stringFrom(fileSet, astFile)
}

func doExtraction(fileSet *token.FileSet, astFile *ast.File, selection Selection, extractedFuncName string) {

	visitor := &astNodeVisitor{parentNode: nil, context: &visitorContext{fset: fileSet, selection: selection}}
	ast.Walk(visitor, astFile)
	if visitor.context.posParent != visitor.context.endParent {
		panic(fmt.Sprintf("Selection is not valid. posParent: %v; endParent: %v",
			visitor.context.posParent, visitor.context.endParent))
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
	var returnTypeString string
	switch context.posParent.(type) {
	case *ast.AssignStmt:
		assignStmt := context.posParent.(*ast.AssignStmt)
		for i, rhs := range assignStmt.Rhs {
			if rhs == extractedExpressionNode {
				assignStmt.Rhs[i] =
					&ast.CallExpr{Fun: &ast.Ident{Name: extractedFuncName}}
			}
		}
		returnTypeString = strings.ToLower(extractedExpressionNode.(*ast.BasicLit).Kind.String())
	case *ast.CallExpr:
		callExpr := context.posParent.(*ast.CallExpr)
		for i, arg := range callExpr.Args {
			if arg == extractedExpressionNode {
				callExpr.Args[i] = &ast.CallExpr{Fun: &ast.Ident{Name: extractedFuncName}}

			}
		}
		returnTypeString = extractedExpressionNode.(*ast.CallExpr).Fun.(*ast.Ident).Obj.Decl.(*ast.FuncDecl).Type.Results.List[0].Type.(*ast.Ident).Name

	// TODO:
	// Add more cases here, e.g. for CallExpr

	default:
		panic(fmt.Sprintf("Type %v not supported yet", reflect.TypeOf(context.posParent)))
	}
	astFile.Decls = append(astFile.Decls, &ast.FuncDecl{
		Name: &ast.Ident{Name: extractedFuncName},
		Type: &ast.FuncType{
			Results: &ast.FieldList{
				List: []*ast.Field{
					&ast.Field{
						Type: &ast.Ident{Name: returnTypeString},
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
