package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"reflect"

	"github.com/pkg/math"

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

func matchExpression(fileSet *token.FileSet, astFile *ast.File, selection Selection) (ast.Expr, ast.Node) {
	visitor := &astNodeVisitorForExpressions{parentNode: nil, context: &expressionVisitorContext{fset: fileSet, selection: selection}}
	ast.Walk(visitor, astFile)
	return visitor.context.exprToExtract, visitor.context.parent
}

func extractExpressionAsFunc(
	astFile *ast.File,
	fileSet *token.FileSet,
	expr ast.Expr,
	parent ast.Node,
	extractedFuncName string) {
	params := varIdentsUsedIn([]ast.Node{expr})
	util.MapStringAstIdentRemoveKeys(params, namesOf(globalVarIdents(astFile)))

	newExpr := CopyNode(callExprWith(extractedFuncName, params)).(ast.Expr)
	RecalcPoses(newExpr, expr.Pos(), nil, 0)
	switch typedNode := parent.(type) {
	case *ast.AssignStmt:
		for i, rhs := range typedNode.Rhs {
			if rhs == expr {
				typedNode.Rhs[i] = newExpr
			}
		}
		for i, lhs := range typedNode.Lhs {
			if lhs == expr {
				typedNode.Lhs[i] = newExpr
			}
		}
	case *ast.CallExpr:
		for i, arg := range typedNode.Args {
			if arg == expr {
				typedNode.Args[i] = newExpr
			}
		}
	case *ast.ExprStmt:
		typedNode.X = newExpr

	case *ast.ReturnStmt:
		for i, result := range typedNode.Results {
			if result == expr {
				typedNode.Results[i] = newExpr
			}
		}
	case *ast.IfStmt:
		if typedNode.Cond == expr {
			typedNode.Cond = newExpr
		}

	case *ast.CaseClause:
		for i, caseExpr := range typedNode.List {
			if caseExpr == expr {
				typedNode.List[i] = newExpr
			}
		}

	case *ast.SwitchStmt:
		if typedNode.Tag == expr {
			typedNode.Tag = newExpr
		}

	case *ast.ForStmt:
		if typedNode.Cond == expr {
			typedNode.Cond = newExpr
		}

	case *ast.RangeStmt:
		if typedNode.Key == expr {
			typedNode.Key = newExpr
		} else if typedNode.Value == expr {
			typedNode.Value = newExpr
		} else if typedNode.X == expr {
			typedNode.X = newExpr
		}

	case *ast.SendStmt:
		if typedNode.Chan == expr {
			typedNode.Chan = newExpr
		} else if typedNode.Value == expr {
			typedNode.Value = newExpr
		}

	case *ast.IncDecStmt:
		if typedNode.X == expr {
			typedNode.X = newExpr
		}

	case *ast.ValueSpec:
		for i, value := range typedNode.Values {
			if value == expr {
				typedNode.Values[i] = newExpr
			}
		}

	default:
		panic(fmt.Sprintf("Type %v not supported yet", reflect.TypeOf(parent)))
	}

	areaRemoved := areaRemoved(fileSet, expr.Pos(), expr.End())
	lineLengths := lineLengthsFrom(fileSet)
	lineNum, numLinesToCut, newLineLength := replacementModifications(fileSet, expr.Pos(), expr.End(), newExpr.End(), lineLengths, areaRemoved)

	shiftPosesAfterPos(astFile, newExpr, expr.End(), newExpr.End()-expr.End())

	singleExprStmtFuncDeclWith := CopyNode(singleExprStmtFuncDeclWith(extractedFuncName, fieldsFrom(params), expr)).(*ast.FuncDecl)
	var moveOffset token.Pos
	RecalcPoses(singleExprStmtFuncDeclWith, token.Pos(math.Max(int(astFile.End()), endOf(astFile.Comments)))+2, &moveOffset, 0)
	astFile.Decls = append(astFile.Decls, singleExprStmtFuncDeclWith)

	areaToBeAppended := insertionModifications(astFile, singleExprStmtFuncDeclWith, areaRemoved)

	lineLengths = append(
		lineLengths[:lineNum+1],
		lineLengths[lineNum+1+numLinesToCut:]...)
	lineLengths[lineNum] = newLineLength
	lineLengths = append(lineLengths, areaToBeAppended...)

	newFileSet := token.NewFileSet()
	newFileSet.AddFile(fileSet.File(1).Name(), 1, sizeFrom(lineLengths))
	success := newFileSet.File(1).SetLines(ConvertLineLengthsToLineOffsets(lineLengths))
	if !success {
		panic("Could not SetLines on File.")
	}
	*fileSet = *newFileSet

	moveComments(astFile, moveOffset, expr.Pos(), expr.End())
}

func sizeFrom(lineLengths []int) (length int) {
	for _, lineLength := range lineLengths {
		length += lineLength + 1
	}
	return
}

func endOf(commentGroups []*ast.CommentGroup) (end int) {
	for _, commentGroup := range commentGroups {
		end = math.Max(endOf2(commentGroup), end)
	}
	return
}

func endOf2(commentGroup *ast.CommentGroup) (end int) {
	for _, comment := range commentGroup.List {
		end = math.Max(int(comment.End()), end)
	}
	return
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
			resetPoses(typeIdent)
			fieldList = append(fieldList, &ast.Field{Type: typeIdent})
		}
		returnType = &ast.FieldList{List: fieldList}
		stmt = &ast.ReturnStmt{Results: []ast.Expr{returnExpr}}
	} else {
		stmt = &ast.ExprStmt{X: returnExpr}
	}

	params := &ast.FieldList{List: fields}
	resetPoses(params)
	return &ast.FuncDecl{
		Name: ast.NewIdent(funcName),
		Type: &ast.FuncType{
			Params:  params,
			Results: returnType,
		},
		Body: &ast.BlockStmt{List: []ast.Stmt{stmt}},
	}
}
