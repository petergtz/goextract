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

type astNodeVisitorForMultipleStatements struct {
	parentNode ast.Node
	context    *visitorContext
}

func (visitor *astNodeVisitorForMultipleStatements) Visit(node ast.Node) (w ast.Visitor) {
	if node != nil {
		if visitor.context.fset.Position(node.Pos()).Line == visitor.context.selection.Begin.Line &&
			visitor.context.fset.Position(node.Pos()).Column == visitor.context.selection.Begin.Column &&
			!visitor.context.shouldRecord {
			fmt.Println("Starting with node at pos", visitor.context.fset.Position(node.Pos()), "and end", visitor.context.fset.Position(node.End()))
			ast.Print(visitor.context.fset, node)
			fmt.Println(node.Pos(), node)
			fmt.Println("Parent")
			ast.Print(visitor.context.fset, visitor.parentNode)
			visitor.context.posParent = visitor.parentNode
			visitor.context.shouldRecord = true
		}
		if visitor.context.shouldRecord && visitor.context.posParent == visitor.parentNode {
			visitor.context.nodesToExtract = append(visitor.context.nodesToExtract, node)
		}
		if visitor.context.fset.Position(node.End()).Line == visitor.context.selection.End.Line &&
			visitor.context.fset.Position(node.End()).Column == visitor.context.selection.End.Column {
			fmt.Println("Ending with node at pos", visitor.context.fset.Position(node.Pos()), "and end", visitor.context.fset.Position(node.End()))
			ast.Print(visitor.context.fset, node)
			fmt.Println("Parent")
			ast.Print(visitor.context.fset, visitor.parentNode)
			visitor.context.endParent = visitor.parentNode
			visitor.context.shouldRecord = false
			return nil
		}
	}
	return &astNodeVisitorForMultipleStatements{
		parentNode: node,
		context:    visitor.context,
	}
}

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
	context := visitor.context
	if len(context.nodesToExtract) == 1 {
		extractExpression(astFile, fileSet, context.nodesToExtract[0].(ast.Expr), context.posParent, extractedFuncName)
	} else {
		v := &astNodeVisitorForMultipleStatements{parentNode: nil, context: &visitorContext{fset: fileSet, selection: selection}}
		ast.Walk(v, astFile)
		if v.context.posParent != v.context.endParent {
			panic(fmt.Sprintf("Selection is not valid. posParent: %v; endParent: %v",
				v.context.posParent, v.context.endParent))
		}
		if v.context.posParent == nil {
			panic(fmt.Sprintf("Selection is not valid. posParent: %v; endParent: %v",
				v.context.posParent, v.context.endParent))
		}
		extractMultipleStatements(astFile, fileSet, v.context, extractedFuncName)
	}
}

func globalVars(astFile *ast.File) []string {
	var result []string
	ast.Inspect(astFile, func(node ast.Node) bool {
		switch typedNode := node.(type) {
		case *ast.FuncDecl:
			return false
		case *ast.GenDecl:
			if typedNode.Tok.String() == "var" {
				for _, spec := range typedNode.Specs {
					for _, name := range spec.(*ast.ValueSpec).Names {
						result = append(result, name.Name)
					}
				}
			}
			return true
		default:
			return true
		}
	})
	return result
}

func varsWithTypesDeclaredWithin(nodes []ast.Node) map[string]string {
	result := make(map[string]string)
	for _, node := range nodes {
		ast.Inspect(node, func(node ast.Node) bool {
			if assignStmt, ok := node.(*ast.AssignStmt); ok && assignStmt.Tok.String() == ":=" {
				for i := range assignStmt.Lhs {
					result[assignStmt.Lhs[i].(*ast.Ident).Name] = deduceTypeString(assignStmt.Rhs[i])
				}
			}
			return true
		})
	}
	return result
}

func allUsedIdentsThatAreVars(nodes []ast.Node) map[string]string {
	result := make(map[string]string)
	for _, node := range nodes {
		ast.Inspect(node, func(node ast.Node) bool {
			if typedNode, ok := node.(*ast.Ident); ok &&
				typedNode.Obj != nil && typedNode.Obj.Kind == ast.Var {
				switch typedDecl := typedNode.Obj.Decl.(type) {
				case *ast.AssignStmt:
					for i, lhs := range typedDecl.Lhs {
						if lhs.(*ast.Ident).Name == typedNode.Name {
							result[typedNode.Name] = deduceTypeString(typedDecl.Rhs[i].(ast.Expr))
						}
					}
				default:
					result[typedNode.Name] = "UnresolvedType"
				}
			}
			return true
		})
	}
	return result
}

func varsWithTypesUsedIn(stmts []ast.Stmt, outOf map[string]string) map[string]string {
	result := make(map[string]string)
	for _, stmt := range stmts {
		ast.Inspect(stmt, func(node ast.Node) bool {
			if ident, ok := node.(*ast.Ident); ok {
				if outOf[ident.Name] != "" {
					result[ident.Name] = outOf[ident.Name]
				}
			}
			return true
		})
	}
	return result
}

func namesOf(vars map[string]string) []string {
	result := make([]string, 0, len(vars))
	for k := range vars {
		result = append(result, k)
	}
	return result
}

func returnExpressionsFrom(vars map[string]string) []ast.Expr {
	var result []ast.Expr
	for k, v := range vars {
		// TODO not sure this is the right way of creating a identifier + type
		result = append(result, &ast.Ident{Name: k, Obj: &ast.Object{Type: ast.NewIdent(v)}})
	}
	return result
}

func extractExprFrom(extractedFuncName string, params map[string]string) *ast.CallExpr {
	return &ast.CallExpr{
		Fun:  ast.NewIdent(extractedFuncName),
		Args: argsFrom(params),
	}
}

func argsFrom(params map[string]string) (result []ast.Expr) {
	for key := range params {
		result = append(result, ast.NewIdent(key))
	}
	return
}

func argsAndTypesFrom(params map[string]string) (result []*ast.Field) {
	for key, val := range params {
		result = append(result, &ast.Field{
			Names: []*ast.Ident{ast.NewIdent(key)},
			Type:  ast.NewIdent(val),
		})
	}
	return
}

func deduceTypes(exprs []ast.Expr) []*ast.Field {
	var result []*ast.Field
	for _, expr := range exprs {
		returnTypeString := deduceTypeString(expr)
		if returnTypeString != "" {
			result = append(result, &ast.Field{Type: ast.NewIdent(returnTypeString)})

		}
	}
	return result
}

func deduceTypeString(expr ast.Expr) string {
	switch typedExpr := expr.(type) {
	case *ast.BasicLit:
		return strings.ToLower(typedExpr.Kind.String())
	case *ast.CallExpr:
		if typedExpr.Fun.(*ast.Ident).Obj.Decl.(*ast.FuncDecl).Type.Results == nil {
			return ""
		}
		result := ""
		for _, res := range typedExpr.Fun.(*ast.Ident).Obj.Decl.(*ast.FuncDecl).Type.Results.List {
			result += " " + res.Type.(*ast.Ident).Name
		}
		return result
	case *ast.Ident:
		// return typedExpr.Obj.Type.(*ast.Ident).Name
		return findTypeFor(typedExpr.Obj.Name, typedExpr.Obj.Decl.(*ast.AssignStmt))
	default:
		return fmt.Sprintf("UnresolvedType_%T", expr)
	}
}

func findTypeFor(name string, assignStmt *ast.AssignStmt) string {
	for i := range assignStmt.Lhs {
		if assignStmt.Lhs[i].(*ast.Ident).Name == name {
			return deduceTypeString(assignStmt.Rhs[i])
		}
	}
	return "UnresolvedType"
}
