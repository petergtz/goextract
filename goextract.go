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
			Type:  deduceTypeIdentForVarIdent(val),
		})
	}
	return
}

func deduceTypeIdentsForExpr(expr ast.Expr) []*ast.Ident {
	switch typedExpr := expr.(type) {
	case *ast.Ident:
		return []*ast.Ident{deduceTypeIdentForVarIdent(typedExpr)}
	case *ast.Ellipsis:
		panic(fmt.Sprintf("Type deduction for %T not implemented yet", expr))
	case *ast.BasicLit:
		return []*ast.Ident{ast.NewIdent(strings.ToLower(typedExpr.Kind.String()))}
	case *ast.FuncLit:
		panic(fmt.Sprintf("Type deduction for %T not implemented yet", expr))
	case *ast.CompositeLit:
		return deduceTypeIdentsForExpr(typedExpr.Type)
	case *ast.ParenExpr:
		panic(fmt.Sprintf("Type deduction for %T not implemented yet", expr))
	case *ast.SelectorExpr:
		panic(fmt.Sprintf("Type deduction for %T not implemented yet", expr))
	case *ast.IndexExpr:
		panic(fmt.Sprintf("Type deduction for %T not implemented yet", expr))
	case *ast.SliceExpr:
		panic(fmt.Sprintf("Type deduction for %T not implemented yet", expr))
	case *ast.TypeAssertExpr:
		panic(fmt.Sprintf("Type deduction for %T not implemented yet", expr))
	case *ast.CallExpr:
		if typedExpr.Fun.(*ast.Ident).Obj.Decl.(*ast.FuncDecl).Type.Results == nil {
			return nil
		}
		var result []*ast.Ident
		for _, res := range typedExpr.Fun.(*ast.Ident).Obj.Decl.(*ast.FuncDecl).Type.Results.List {
			result = append(result, res.Type.(*ast.Ident))
		}
		return result
	case *ast.StarExpr:
		panic(fmt.Sprintf("Type deduction for %T not implemented yet", expr))
	case *ast.UnaryExpr:
		if typedExpr.Op == token.RANGE {
			ast.Print(nil, typedExpr)
			return []*ast.Ident{ast.NewIdent("int"), deduceTypeIdentsForExpr(typedExpr.X)[0]}
		} else {

			panic("UnaryExpr not implemented yet")
		}
	case *ast.BinaryExpr:
		panic(fmt.Sprintf("Type deduction for %T not implemented yet", expr))
	case *ast.KeyValueExpr:
		panic(fmt.Sprintf("Type deduction for %T not implemented yet", expr))
	case *ast.ArrayType:
		return []*ast.Ident{typedExpr.Elt.(*ast.Ident)}
	case *ast.StructType:
		panic(fmt.Sprintf("Type deduction for %T not implemented yet", expr))
	case *ast.FuncType:
		panic(fmt.Sprintf("Type deduction for %T not implemented yet", expr))
	case *ast.InterfaceType:
		panic(fmt.Sprintf("Type deduction for %T not implemented yet", expr))
	case *ast.MapType:
		panic(fmt.Sprintf("Type deduction for %T not implemented yet", expr))
	case *ast.ChanType:
		panic(fmt.Sprintf("Type deduction for %T not implemented yet", expr))

	default:
		panic(fmt.Sprintf("Type deduction for %T not implemented yet", expr))
	}
}

func deduceTypeIdentForVarIdent(ident *ast.Ident) *ast.Ident {
	if ident.Obj.Kind != ast.Var {
		panic("Expected var type for ident")
	}
	switch typedDecl := ident.Obj.Decl.(type) {
	case *ast.AssignStmt:
		for i, lhs := range typedDecl.Lhs {
			if lhs.(*ast.Ident).Obj == ident.Obj {
				if len(typedDecl.Rhs) == 0 {
					panic("Unexpected empty Rhs")
				}
				if len(typedDecl.Rhs) == 1 {
					return deduceTypeIdentsForExpr(typedDecl.Rhs[0])[i]
				} else {
					return deduceTypeIdentsForExpr(typedDecl.Rhs[i])[0]
				}
			}
		}
		panic("Unexpected: no result in AssignStmt")
	case *ast.ValueSpec:
		for i, name := range typedDecl.Names {
			if name.Obj == ident.Obj {
				if typedDecl.Type != nil {
					return typedDecl.Type.(*ast.Ident)
				} else {
					if len(typedDecl.Values) == 0 {
						panic("Unexpected empty value")
					}
					if len(typedDecl.Values) == 1 {
						return deduceTypeIdentsForExpr(typedDecl.Values[0])[i]
					} else {
						return deduceTypeIdentsForExpr(typedDecl.Values[i])[0]
					}

				}
			}
		}
		panic("Unexpected: no result in ValueSpec")
	default:
		panic("Unexpected decl type")
	}
}

func deduceTypeIdentsForVarIdents(varIdents []*ast.Ident) (typeIdents []*ast.Ident) {
	for _, ident := range varIdents {
		typeIdents = append(typeIdents, deduceTypeIdentForVarIdent(ident))
	}
	return
}
