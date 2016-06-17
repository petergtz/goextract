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
	"os"
	"os/exec"
	"strings"

	"github.com/petergtz/goextract/util"
)

func ExtractFileToFile(inputFileName string, selection Selection, extractedFuncName string, outputFilename string) {
	fileSet, astFile := astFromFile(inputFileName)
	createAstFileDump(inputFileName+".ast", fileSet, astFile)
	doExtraction(fileSet, astFile, selection, extractedFuncName)
	util.WriteFileAsStringOrPanic(outputFilename, stringFrom(fileSet, astFile))
	err := exec.Command("gofmt", "-w", outputFilename).Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, util.ReadFileAsStringOrPanic(outputFilename))
		panic(err)
	}
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
	visitor := &astNodeVisitorForExpressions{parentNode: nil, context: &expressionVisitorContext{fset: fileSet, selection: selection}}
	ast.Walk(visitor, astFile)
	if visitor.context.exprToExtract != nil {
		extractExpression(astFile, fileSet, visitor.context.exprToExtract, visitor.context.parent, extractedFuncName)
	} else {
		v := &astNodeVisitorForMultipleStatements{parentNode: nil, context: &multipleStatementVisitorContext{fset: fileSet, selection: selection}}
		ast.Walk(v, astFile)
		if v.context.posParent != v.context.endParent {
			panic(fmt.Sprintf("Selection is not valid. posParent: %v; endParent: %v",
				v.context.posParent, v.context.endParent))
		}
		if v.context.posParent == nil {
			panic(fmt.Sprintf("Selection is not valid. posParent: %v; endParent: %v",
				v.context.posParent, v.context.endParent))
		}
		extractMultipleStatements(astFile, fileSet, v.context.nodesToExtract, v.context.posParent, extractedFuncName)
	}
}

func globalVarIdents(astFile *ast.File) map[string]*ast.Ident {
	result := make(map[string]*ast.Ident)
	ast.Inspect(astFile, func(node ast.Node) bool {
		switch typedNode := node.(type) {
		case *ast.FuncDecl:
			return false
		case *ast.GenDecl:
			if typedNode.Tok.String() == "var" {
				for _, spec := range typedNode.Specs {
					for _, name := range spec.(*ast.ValueSpec).Names {
						result[name.Name] = name
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

func varIdentsDeclaredWithin(nodes []ast.Node) map[string]*ast.Ident {
	result := make(map[string]*ast.Ident)
	for _, node := range nodes {
		ast.Inspect(node, func(node ast.Node) bool {
			if assignStmt, ok := node.(*ast.AssignStmt); ok && assignStmt.Tok.String() == ":=" {
				for i := range assignStmt.Lhs {
					result[assignStmt.Lhs[i].(*ast.Ident).Name] = assignStmt.Lhs[i].(*ast.Ident)
				}
			}
			return true
		})
	}
	return result
}

func varIdentsUsedIn(nodes []ast.Node) map[string]*ast.Ident {
	result := make(map[string]*ast.Ident)
	for _, node := range nodes {
		ast.Inspect(node, func(node ast.Node) bool {
			if ident, ok := node.(*ast.Ident); ok &&
				ident.Obj != nil && ident.Obj.Kind == ast.Var {
				switch typedDecl := ident.Obj.Decl.(type) {
				case *ast.AssignStmt:
					for _, lhs := range typedDecl.Lhs {
						if lhs.(*ast.Ident).Name == ident.Name {
							result[ident.Name] = ident
						}
					}
				default:
					result[ident.Name] = ast.NewIdent("UnresolvedType")
				}
			}
			return true
		})
	}
	return result
}

// TODO rename to varIdentsUsedIn
func overlappingVarsIdentsUsedIn(stmts []ast.Stmt, outOf map[string]*ast.Ident) map[string]*ast.Ident {
	result := make(map[string]*ast.Ident)
	for _, stmt := range stmts {
		ast.Inspect(stmt, func(node ast.Node) bool {
			if ident, ok := node.(*ast.Ident); ok {
				if outOf[ident.Name] != nil {
					result[ident.Name] = ident
				}
			}
			return true
		})
	}
	return result
}

func namesOf(idents map[string]*ast.Ident) []string {
	result := make([]string, 0, len(idents))
	for k := range idents {
		result = append(result, k)
	}
	return result
}

func exprsFrom(idents map[string]*ast.Ident) []ast.Expr {
	result := make([]ast.Expr, 0, len(idents))
	for _, v := range idents {
		result = append(result, v)
	}
	return result
}

func callExprWith(funcName string, params map[string]*ast.Ident) *ast.CallExpr {
	return &ast.CallExpr{
		Fun:  ast.NewIdent(funcName),
		Args: exprsFrom(params),
	}
}

func fieldsFrom(params map[string]*ast.Ident) (result []*ast.Field) {
	for key, val := range params {
		result = append(result, &ast.Field{
			Names: []*ast.Ident{ast.NewIdent(key)},
			Type:  ast.NewIdent(deduceTypeString(val)),
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

func deduceTypeIdentsForExpr(expr ast.Expr) []*ast.Ident {

}

func deduceTypeIdentForVarIdent(ident *ast.Ident) *ast.Ident {

}

func deduceTypeIdentsForVarIdents(ident []*ast.Ident) []*ast.Ident {

}

func deduceTypeString(expr ast.Expr) string {
	switch typedExpr := expr.(type) {
	case *ast.BasicLit:
		return strings.ToLower(typedExpr.Kind.String())
	case *ast.CallExpr:
		if typedExpr.Fun.(*ast.Ident).Obj.Decl.(*ast.FuncDecl).Type.Results == nil {
			return ""
		}
		var result []string
		for _, res := range typedExpr.Fun.(*ast.Ident).Obj.Decl.(*ast.FuncDecl).Type.Results.List {
			result = append(result, res.Type.(*ast.Ident).Name)
		}
		return "(" + strings.Join(result, ", ") + ")"
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
			ast.Print(nil, assignStmt)
			return deduceTypeString(assignStmt.Rhs[i])
		}
	}
	return "UnresolvedType"
}
