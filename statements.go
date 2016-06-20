package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"reflect"

	"github.com/petergtz/goextract/util"
)

type multipleStatementVisitorContext struct {
	fset           *token.FileSet
	posParent      ast.Node
	endParent      ast.Node
	nodesToExtract []ast.Node
	shouldRecord   bool

	selection Selection
}

type astNodeVisitorForMultipleStatements struct {
	parentNode ast.Node
	context    *multipleStatementVisitorContext
}

func (visitor *astNodeVisitorForMultipleStatements) Visit(node ast.Node) (w ast.Visitor) {
	if node != nil {
		if visitor.context.fset.Position(node.Pos()).Line == visitor.context.selection.Begin.Line &&
			visitor.context.fset.Position(node.Pos()).Column == visitor.context.selection.Begin.Column &&
			!visitor.context.shouldRecord {
			// fmt.Println("Starting with node at pos", visitor.context.fset.Position(node.Pos()), "and end", visitor.context.fset.Position(node.End()))
			// ast.Print(visitor.context.fset, node)
			// fmt.Println(node.Pos(), node)
			// fmt.Println("Parent")
			// ast.Print(visitor.context.fset, visitor.parentNode)
			visitor.context.posParent = visitor.parentNode
			visitor.context.shouldRecord = true
		}
		if visitor.context.shouldRecord && visitor.context.posParent == visitor.parentNode {
			visitor.context.nodesToExtract = append(visitor.context.nodesToExtract, node)
		}
		if visitor.context.fset.Position(node.End()).Line == visitor.context.selection.End.Line &&
			visitor.context.fset.Position(node.End()).Column == visitor.context.selection.End.Column {
			// fmt.Println("Ending with node at pos", visitor.context.fset.Position(node.Pos()), "and end", visitor.context.fset.Position(node.End()))
			// ast.Print(visitor.context.fset, node)
			// fmt.Println("Parent")
			// ast.Print(visitor.context.fset, visitor.parentNode)
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

func extractMultipleStatements(
	astFile *ast.File,
	fileSet *token.FileSet,
	stmtsToExtract []ast.Node,
	parentNode ast.Node,
	extractedFuncName string) {
	params := varIdentsUsedIn(stmtsToExtract)
	varsDeclaredWithinStmtsToExtract := varIdentsDeclaredWithin(stmtsToExtract)
	util.MapStringAstIdentRemoveKeys(params, namesOf(varsDeclaredWithinStmtsToExtract))
	util.MapStringAstIdentRemoveKeys(params, namesOf(globalVarIdents(astFile)))

	allStmts := stmtsFromBlockStmt(parentNode)
	indexOfExtractedStmt := indexOf(stmtsToExtract[0].(ast.Stmt), *allStmts)
	varsUsedAfterwards := overlappingVarsIdentsUsedIn((*allStmts)[indexOfExtractedStmt+len(stmtsToExtract):], varsDeclaredWithinStmtsToExtract)

	replaceStmtsWithFuncCallStmt(
		allStmts,
		indexOfExtractedStmt, len(stmtsToExtract),
		varsUsedAfterwards, extractedFuncName, params)

	astFile.Decls = append(astFile.Decls, multipleStmtFuncDeclWith(
		extractedFuncName,
		fieldsFrom(params),
		stmtsFromNodes(stmtsToExtract),
		exprsFrom(varsUsedAfterwards),
	))
}

func stmtsFromBlockStmt(node ast.Node) *[]ast.Stmt {
	switch typedNode := node.(type) {
	case *ast.BlockStmt:
		return &typedNode.List
	case *ast.CaseClause:
		return &typedNode.Body
	case *ast.CommClause:
		return &typedNode.Body
	default:
		panic(fmt.Sprintf("Type %v not supported yet", reflect.TypeOf(node)))
	}
}

func replaceStmtsWithFuncCallStmt(allStmts *[]ast.Stmt, indexOfExtractedStmt int, count int, varsUsedAfterwards map[string]*ast.Ident, extractedFuncName string, params map[string]*ast.Ident) {
	(*allStmts)[indexOfExtractedStmt] = funcCallStmt(varsUsedAfterwards, extractedFuncName, params)
	(*allStmts) = append((*allStmts)[:indexOfExtractedStmt+1], (*allStmts)[indexOfExtractedStmt+count:]...)
}

func funcCallStmt(varsUsedAfterwards map[string]*ast.Ident, extractedFuncName string, params map[string]*ast.Ident) ast.Stmt {
	if len(varsUsedAfterwards) == 0 {
		return &ast.ExprStmt{X: callExprWith(extractedFuncName, params)}
	} else {
		return &ast.AssignStmt{
			Lhs: []ast.Expr{ast.NewIdent(namesOf(varsUsedAfterwards)[0])},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{callExprWith(extractedFuncName, params)},
		}
	}
}

func identsFromExprs(exprs []ast.Expr) (idents []*ast.Ident) {
	for _, expr := range exprs {
		idents = append(idents, expr.(*ast.Ident))
	}
	return
}

func multipleStmtFuncDeclWith(
	extractedFuncName string,
	fields []*ast.Field,
	stmts []ast.Stmt,
	// TODO should these be ast.Idents?
	definedVars []ast.Expr) *ast.FuncDecl {

	allStmts := make([]ast.Stmt, len(stmts), len(stmts)+1)
	copy(allStmts, stmts)
	var returnType *ast.FieldList
	if len(definedVars) != 0 {
		allStmts = append(allStmts, &ast.ReturnStmt{Results: definedVars})
		returnType = &ast.FieldList{List: fieldListFromIdents(definedVars)}
	}
	return &ast.FuncDecl{
		Name: ast.NewIdent(extractedFuncName),
		Type: &ast.FuncType{
			Params:  &ast.FieldList{List: fields},
			Results: returnType,
		},
		Body: &ast.BlockStmt{List: allStmts},
	}
}

func indexOf(stmtToFind ast.Stmt, stmts []ast.Stmt) int {
	for i, stmt := range stmts {
		if stmt == stmtToFind {
			return i
		}
	}
	panic("Unexpected: statement not in list")
}

func stmtsFromNodes(nodes []ast.Node) []ast.Stmt {
	stmts := make([]ast.Stmt, len(nodes))
	for i, node := range nodes {
		stmts[i] = node.(ast.Stmt)
	}
	return stmts
}

func fieldListFromIdents(idents []ast.Expr) []*ast.Field {
	var fieldList []*ast.Field
	for _, typeIdent := range deduceTypeExprsForVarIdents(identsFromExprs(idents)) {
		fieldList = append(fieldList, &ast.Field{Type: typeIdent})
	}
	return fieldList
}
