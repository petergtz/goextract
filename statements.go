package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"reflect"

	"github.com/petergtz/goextract/util"
)

func extractMultipleStatements(
	astFile *ast.File,
	fileSet *token.FileSet,
	context *visitorContext,
	extractedFuncName string) {
	params := allUsedIdentsThatAreVars(context.nodesToExtract)
	varsDeclaredWithin := varsWithTypesDeclaredWithin(context.nodesToExtract)
	util.MapStringStringRemoveKeys(params, namesOf(varsDeclaredWithin))
	util.MapStringStringRemoveKeys(params, globalVars(astFile))

	var varsUsedAfterwards map[string]string

	var stmts []ast.Stmt

	switch typedNode := context.posParent.(type) {
	case *ast.BlockStmt:
		var indexOfExtractedStmt int
		for i, stmt := range typedNode.List {
			if stmt == context.nodesToExtract[0] {
				indexOfExtractedStmt = i
				break
			}
		}
		varsUsedAfterwards = varsWithTypesUsedIn(typedNode.List[indexOfExtractedStmt+len(context.nodesToExtract):], varsDeclaredWithin)
		for _, node := range context.nodesToExtract {
			stmts = append(stmts, node.(ast.Stmt))
		}
		if len(varsUsedAfterwards) == 0 {
			typedNode.List[indexOfExtractedStmt] = &ast.ExprStmt{X: extractExprFrom(extractedFuncName, params)}

		} else {
			typedNode.List[indexOfExtractedStmt] = &ast.AssignStmt{
				Lhs: []ast.Expr{ast.NewIdent(namesOf(varsUsedAfterwards)[0])},
				Tok: token.DEFINE,
				Rhs: []ast.Expr{extractExprFrom(extractedFuncName, params)},
			}

		}
		typedNode.List = append(typedNode.List[:indexOfExtractedStmt+1], typedNode.List[indexOfExtractedStmt+len(context.nodesToExtract):]...)

	// TODO: Add cases for CommClause and CaseClause here

	default:
		panic(fmt.Sprintf("Type %v not supported yet", reflect.TypeOf(context.posParent)))
	}

	insertExtractedStmtsFuncInto(
		astFile,
		extractedFuncName,
		argsAndTypesFrom(params),
		stmts,
		returnExpressionsFrom(varsUsedAfterwards),
	)
}

func insertExtractedStmtsFuncInto(
	astFile *ast.File,
	extractedFuncName string,
	argsAndTypes []*ast.Field,
	stmts []ast.Stmt,
	definedVars []ast.Expr) {

	allStmts := make([]ast.Stmt, len(stmts), len(stmts)+1)
	copy(allStmts, stmts)
	var returnType *ast.FieldList
	if len(definedVars) != 0 {
		allStmts = append(allStmts, &ast.ReturnStmt{Results: definedVars})
		returnType = &ast.FieldList{List: deduceTypes(definedVars)}
	}
	astFile.Decls = append(astFile.Decls, &ast.FuncDecl{
		Name: ast.NewIdent(extractedFuncName),
		Type: &ast.FuncType{
			Params:  &ast.FieldList{List: argsAndTypes},
			Results: returnType,
		},
		Body: &ast.BlockStmt{List: allStmts},
	})
}
