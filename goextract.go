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
	"os/exec"
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

type astNodeVisitorForExpressions struct {
	parentNode ast.Node
	context    *visitorContext
}

func (visitor *astNodeVisitorForExpressions) Visit(node ast.Node) (w ast.Visitor) {
	if node != nil {
		if visitor.context.fset.Position(node.Pos()).Line == visitor.context.selection.Begin.Line &&
			visitor.context.fset.Position(node.Pos()).Column == visitor.context.selection.Begin.Column &&
			visitor.context.fset.Position(node.End()).Line == visitor.context.selection.End.Line &&
			visitor.context.fset.Position(node.End()).Column == visitor.context.selection.End.Column {
			// fmt.Println("Starting with node at pos", visitor.context.fset.Position(node.Pos()), "and end", visitor.context.fset.Position(node.End()))
			// ast.Print(visitor.context.fset, node)
			// fmt.Println(node.Pos(), node)
			visitor.context.posParent = visitor.parentNode
			visitor.context.endParent = visitor.parentNode
			visitor.context.nodesToExtract = append(visitor.context.nodesToExtract, node)
			return nil
		}
	}
	return &astNodeVisitorForExpressions{
		parentNode: node,
		context:    visitor.context,
	}
}

// func (visitor *astNodeVisitorForMultipleStatements) Visit(node ast.Node) (w ast.Visitor) {
// 	if node != nil {
// 		if visitor.context.fset.Position(node.Pos()).Line == visitor.context.selection.Begin.Line &&
// 			visitor.context.fset.Position(node.Pos()).Column == visitor.context.selection.Begin.Column {
// 			fmt.Println("Starting with node at pos", visitor.context.fset.Position(node.Pos()), "and end", visitor.context.fset.Position(node.End()))
// 			ast.Print(visitor.context.fset, node)
// 			fmt.Println(node.Pos(), node)
// 			visitor.context.posParent = visitor.parentNode
// 			visitor.context.shouldRecord = true
// 		}
// 		if visitor.context.shouldRecord {
// 			visitor.context.nodesToExtract = append(visitor.context.nodesToExtract, node)
// 		}
// 		if visitor.context.fset.Position(node.End()).Line == visitor.context.selection.End.Line &&
// 			visitor.context.fset.Position(node.End()).Column == visitor.context.selection.End.Column {
// 			fmt.Println("Ending with node at pos", visitor.context.fset.Position(node.Pos()), "and end", visitor.context.fset.Position(node.End()))
// 			ast.Print(visitor.context.fset, node)
// 			visitor.context.endParent = visitor.parentNode
// 			visitor.context.shouldRecord = false
// 			return nil
// 		}
// 	}
// 	return &astNodeVisitor{
// 		parentNode: node,
// 		context:    visitor.context,
// 	}
// }

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
	err := exec.Command("gofmt", "-w", outputFilename).Run()
	util.PanicOnError(err)
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

	visitor := &astNodeVisitorForExpressions{parentNode: nil, context: &visitorContext{fset: fileSet, selection: selection}}
	ast.Walk(visitor, astFile)
	if visitor.context.posParent != visitor.context.endParent {
		panic(fmt.Sprintf("Selection is not valid. posParent: %v; endParent: %v",
			visitor.context.posParent, visitor.context.endParent))
	}
	if len(visitor.context.nodesToExtract) == 1 {
		extractExpression(astFile, fileSet, visitor.context, extractedFuncName)
	} else {
		panic("Only expression extractions are supported so far.")
	}

}

type varListerVisitor struct {
	fileSet *token.FileSet
	vars    map[string]string
}

func (visitor *varListerVisitor) Visit(node ast.Node) (w ast.Visitor) {
	if typedNode, ok := node.(*ast.Ident); ok && typedNode.Obj.Kind == ast.Var {
		var typeString string
		switch typedDecl := typedNode.Obj.Decl.(type) {
		case *ast.AssignStmt:
			for i, lhs := range typedDecl.Lhs {
				if lhs.(*ast.Ident).Name == typedNode.Name {
					typeString = deduceReturnTypeString(typedDecl.Rhs[i].(ast.Expr))
				}
			}
		default:
			typeString = "UnresolvedType"
		}
		visitor.vars[typedNode.Name] = typeString
	}
	return visitor
}

func listAllUsedIdentifiersThatAreVars(node ast.Node, fileSet *token.FileSet) map[string]string {
	v := &varListerVisitor{fileSet: fileSet, vars: make(map[string]string)}
	ast.Walk(v, node)
	return v.vars
}

func extractExpression(astFile *ast.File, fileSet *token.FileSet, context *visitorContext, extractedFuncName string) {
	extractedExpressionNode := context.nodesToExtract[0].(ast.Expr)

	// TODO: Ideally this would only list variables that are not available
	// outside of the scope where the expressions lives
	params := listAllUsedIdentifiersThatAreVars(extractedExpressionNode, fileSet)

	extractExpr := &ast.CallExpr{Fun: &ast.Ident{Name: extractedFuncName}, Args: argsFrom(params)}
	switch typedNode := context.posParent.(type) {
	case *ast.AssignStmt:
		for i, rhs := range typedNode.Rhs {
			if rhs == extractedExpressionNode {
				typedNode.Rhs[i] = extractExpr
			}
		}
	case *ast.CallExpr:
		for i, arg := range typedNode.Args {
			if arg == extractedExpressionNode {
				typedNode.Args[i] = extractExpr
			}
		}
	// TODO:
	// Add more cases here

	default:
		panic(fmt.Sprintf("Type %v not supported yet", reflect.TypeOf(context.posParent)))
	}

	insertExtractedFuncInto(astFile, extractedFuncName, argsAndTypesFrom(params), extractedExpressionNode)
}

func argsFrom(params map[string]string) (result []ast.Expr) {
	for key := range params {
		result = append(result, &ast.Ident{Name: key})
	}
	return
}

func argsAndTypesFrom(params map[string]string) (result []*ast.Field) {
	for key, val := range params {
		result = append(result, &ast.Field{Names: []*ast.Ident{&ast.Ident{Name: key}}, Type: &ast.Ident{Name: val}})
	}
	return
}

func insertExtractedFuncInto(astFile *ast.File, extractedFuncName string, argsAndTypes []*ast.Field, extractedExpressionNode ast.Expr) {
	astFile.Decls = append(astFile.Decls, &ast.FuncDecl{
		Name: &ast.Ident{Name: extractedFuncName},

		Type: &ast.FuncType{
			Params: &ast.FieldList{
				List: argsAndTypes,
			},
			Results: &ast.FieldList{
				List: []*ast.Field{
					&ast.Field{
						Type: &ast.Ident{Name: deduceReturnTypeString(extractedExpressionNode)},
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

func deduceReturnTypeString(expr ast.Expr) string {
	switch typedExpr := expr.(type) {
	case *ast.BasicLit:
		return strings.ToLower(typedExpr.Kind.String())
	case *ast.CallExpr:
		return typedExpr.Fun.(*ast.Ident).Obj.Decl.(*ast.FuncDecl).Type.Results.List[0].Type.(*ast.Ident).Name
	default:
		return "TODO"
	}
}
