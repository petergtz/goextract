package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"reflect"

	"github.com/petergtz/goextract/util"
)

type astNodeVisitorForExpressions struct {
	parentNode ast.Node
	context    *visitorContext
}

func (visitor *astNodeVisitorForExpressions) Visit(node ast.Node) (w ast.Visitor) {
	if node != nil {
		_, isExpr := node.(ast.Expr)
		if visitor.context.fset.Position(node.Pos()).Line == visitor.context.selection.Begin.Line &&
			visitor.context.fset.Position(node.Pos()).Column == visitor.context.selection.Begin.Column &&
			visitor.context.fset.Position(node.End()).Line == visitor.context.selection.End.Line &&
			visitor.context.fset.Position(node.End()).Column == visitor.context.selection.End.Column &&
			isExpr {
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

func extractExpression(
	astFile *ast.File,
	fileSet *token.FileSet,
	expr ast.Expr,
	parent ast.Node,
	extractedFuncName string) {
	params := allUsedIdentsThatAreVars([]ast.Node{expr})
	util.MapStringStringRemoveKeys(params, globalVars(astFile))

	switch typedNode := parent.(type) {
	case *ast.AssignStmt:
		for i, rhs := range typedNode.Rhs {
			if rhs == expr {
				typedNode.Rhs[i] = extractExprFrom(extractedFuncName, params)
			}
		}
	case *ast.CallExpr:
		for i, arg := range typedNode.Args {
			if arg == expr {
				typedNode.Args[i] = extractExprFrom(extractedFuncName, params)
			}
		}
	case *ast.ExprStmt:
		typedNode.X = extractExprFrom(extractedFuncName, params)

	case *ast.ReturnStmt:
		for i, result := range typedNode.Results {
			if result == expr {
				typedNode.Results[i] = extractExprFrom(extractedFuncName, params)
			}
		}

	// TODO:
	// Add more cases here

	default:
		panic(fmt.Sprintf("Type %v not supported yet", reflect.TypeOf(parent)))
	}
	insertExtractedExpressionFuncInto(
		astFile,
		extractedFuncName,
		argsAndTypesFrom(params),
		expr)
}

func insertExtractedExpressionFuncInto(
	astFile *ast.File,
	extractedFuncName string,
	argsAndTypes []*ast.Field,
	returnExpr ast.Expr) {

	var returnType *ast.FieldList
	returnTypeString := deduceTypeString(returnExpr)
	var stmt ast.Stmt
	if returnTypeString != "" {
		returnType = &ast.FieldList{List: []*ast.Field{&ast.Field{Type: ast.NewIdent(returnTypeString)}}}
		stmt = &ast.ReturnStmt{Results: []ast.Expr{returnExpr}}
	} else {
		stmt = &ast.ExprStmt{X: returnExpr}
	}

	astFile.Decls = append(astFile.Decls, &ast.FuncDecl{
		Name: ast.NewIdent(extractedFuncName),
		Type: &ast.FuncType{
			Params:  &ast.FieldList{List: argsAndTypes},
			Results: returnType,
		},
		Body: &ast.BlockStmt{List: []ast.Stmt{stmt}},
	})
}
