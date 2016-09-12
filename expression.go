package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"reflect"
	"strings"

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

	pkg, info := typesPackage(astFile, fileSet)

	params := varIdentsUsedIn([]ast.Node{expr})
	util.MapStringAstIdentRemoveKeys(params, namesOf(globalVarIdents(astFile)))

	extractionPos := expr.Pos()
	extractionEnd := expr.End()
	newExpr := callExprWith(extractedFuncName, params, extractionPos)
	replaceExprWithCallExpr(parent, expr, newExpr)

	removedComments := removeComments(astFile, extractionPos, extractionEnd)

	shiftPosesAfterPos(astFile, newExpr, extractionEnd, newExpr.End()-extractionEnd)

	singleExprStmtFuncDecl, moveOffset := singleExprStmtFuncDeclWith(
		extractedFuncName,
		fieldsFrom(params, pkg),
		expr,
		info,
		token.Pos(math.Max(int(astFile.End()), endOf(astFile.Comments)))+2,
	)
	astFile.Comments = append(astFile.Comments, removedComments...)
	astFile.Decls = append(astFile.Decls, singleExprStmtFuncDecl)

	moveComments(astFile, moveOffset, extractionPos, extractionEnd)

	areaRemoved := areaRemoved(fileSet, extractionPos, extractionEnd)
	areaToBeAppended := areaToBeAppendedForExpr(singleExprStmtFuncDecl, areaRemoved)
	lineLengths := recalcLineLengths(lineLengthsFrom(fileSet), fileSet, extractionPos, extractionEnd, newExpr.End(), areaRemoved, areaToBeAppended)

	newFileSet := token.NewFileSet()
	newFileSet.AddFile(fileSet.File(1).Name(), 1, sizeFrom(lineLengths))
	success := newFileSet.File(1).SetLines(ConvertLineLengthsToLineOffsets(lineLengths))
	if !success {
		panic("Could not SetLines on File.")
	}
	*fileSet = *newFileSet
}

func replaceExprWithCallExpr(parent ast.Node, expr ast.Expr, newExpr ast.Expr) {
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

}

func removeComments(astFile *ast.File, pos, end token.Pos) (result []*ast.CommentGroup) {
	for i, commentGroup := range astFile.Comments {
		for _, comment := range commentGroup.List {
			if comment.Slash >= pos && comment.Slash <= end {
				result = append(result, commentGroup)
				astFile.Comments = append(astFile.Comments[:i], astFile.Comments[i+1:]...)
				break
			}
		}
	}
	return
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

func singleExprStmtFuncDeclWith(funcName string, fields []*ast.Field, returnExpr ast.Expr, info *types.Info, pos token.Pos) (*ast.FuncDecl, token.Pos) {
	var (
		returnType *ast.FieldList
		stmt       ast.Stmt
	)
	typeIdents := []ast.Expr{}
	typ := info.TypeOf(returnExpr)
	if typ != nil {
		typeString := typ.String()
		if strings.HasPrefix(typeString, "(") && strings.HasSuffix(typeString, ")") {
			for _, s := range strings.Split(typeString[1:len(typeString)-1], ",") {
				if s != "" {
					typeIdents = append(typeIdents, ast.NewIdent(s))
				}
			}
		} else {
			typeIdents = []ast.Expr{ast.NewIdent(strings.TrimPrefix(typeString, "untyped "))}
		}
	}
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
	result := CopyNode(&ast.FuncDecl{
		Name: ast.NewIdent(funcName),
		Type: &ast.FuncType{
			Params:  params,
			Results: returnType,
		},
		Body: &ast.BlockStmt{List: []ast.Stmt{stmt}},
	}).(*ast.FuncDecl)
	var moveOffset token.Pos
	RecalcPoses(result, pos, &moveOffset, 0)
	return result, moveOffset
}
