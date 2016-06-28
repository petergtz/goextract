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
	context    *expressionVisitorContext
}

type expressionVisitorContext struct {
	fset          *token.FileSet
	parent        ast.Node
	exprToExtract ast.Expr

	selection Selection
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
			visitor.context.parent = visitor.parentNode
			visitor.context.exprToExtract = node.(ast.Expr)
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
	params := varIdentsUsedIn([]ast.Node{expr})
	util.MapStringAstIdentRemoveKeys(params, namesOf(globalVarIdents(astFile)))

	switch typedNode := parent.(type) {
	case *ast.AssignStmt:
		for i, rhs := range typedNode.Rhs {
			if rhs == expr {
				typedNode.Rhs[i] = callExprWith(extractedFuncName, params)
			}
		}
		for i, lhs := range typedNode.Lhs {
			if lhs == expr {
				typedNode.Lhs[i] = callExprWith(extractedFuncName, params)
			}
		}
	case *ast.CallExpr:
		for i, arg := range typedNode.Args {
			if arg == expr {
				typedNode.Args[i] = callExprWith(extractedFuncName, params)
			}
		}
	case *ast.ExprStmt:
		typedNode.X = callExprWith(extractedFuncName, params)

	case *ast.ReturnStmt:
		for i, result := range typedNode.Results {
			if result == expr {
				typedNode.Results[i] = callExprWith(extractedFuncName, params)
			}
		}
	case *ast.IfStmt:
		if typedNode.Cond == expr {
			typedNode.Cond = callExprWith(extractedFuncName, params)
		}

	case *ast.CaseClause:
		for i, caseExpr := range typedNode.List {
			if caseExpr == expr {
				typedNode.List[i] = callExprWith(extractedFuncName, params)
			}
		}

	case *ast.SwitchStmt:
		if typedNode.Tag == expr {
			typedNode.Tag = callExprWith(extractedFuncName, params)
		}

	case *ast.ForStmt:
		if typedNode.Cond == expr {
			typedNode.Cond = callExprWith(extractedFuncName, params)
		}

	case *ast.RangeStmt:
		if typedNode.Key == expr {
			typedNode.Key = callExprWith(extractedFuncName, params)
		} else if typedNode.Value == expr {
			typedNode.Value = callExprWith(extractedFuncName, params)
		} else if typedNode.X == expr {
			typedNode.X = callExprWith(extractedFuncName, params)
		}

	case *ast.SendStmt:
		if typedNode.Chan == expr {
			typedNode.Chan = callExprWith(extractedFuncName, params)
		} else if typedNode.Value == expr {
			typedNode.Value = callExprWith(extractedFuncName, params)
		}

	case *ast.IncDecStmt:
		if typedNode.X == expr {
			typedNode.X = callExprWith(extractedFuncName, params)
		}

	default:
		panic(fmt.Sprintf("Type %v not supported yet", reflect.TypeOf(parent)))
	}

	astFile.Decls = append(astFile.Decls, singleExprStmtFuncDeclWith(
		extractedFuncName,
		fieldsFrom(params),
		expr))
}

func singleExprStmtFuncDeclWith(funcName string, fields []*ast.Field, returnExpr ast.Expr) *ast.FuncDecl {
	var (
		returnType *ast.FieldList
		stmt       ast.Stmt
	)
	typeIdents := deduceTypeExprsForExpr(returnExpr)
	if len(typeIdents) != 0 {
		var fieldList []*ast.Field
		for _, typeIdent := range typeIdents {
			fieldList = append(fieldList, &ast.Field{Type: typeIdent})
		}
		returnType = &ast.FieldList{List: fieldList}
		stmt = &ast.ReturnStmt{Results: []ast.Expr{returnExpr}}
	} else {
		stmt = &ast.ExprStmt{X: returnExpr}
	}

	return &ast.FuncDecl{
		Name: ast.NewIdent(funcName),
		Type: &ast.FuncType{
			Params:  &ast.FieldList{List: fields},
			Results: returnType,
		},
		Body: &ast.BlockStmt{List: []ast.Stmt{stmt}},
	}
}
