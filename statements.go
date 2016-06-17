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

func extractMultipleStatements(
	astFile *ast.File,
	fileSet *token.FileSet,
	stmtsToExtract []ast.Node,
	parentNode ast.Node,
	extractedFuncName string) {
	params := allUsedIdentsThatAreVars(stmtsToExtract)
	varsDeclaredWithinStmtsToExtract :=
		varsWithTypesDeclaredWithin(stmtsToExtract)
	util.MapStringAstIdentRemoveKeys(params, namesOf(varsDeclaredWithinStmtsToExtract))
	util.MapStringAstIdentRemoveKeys(params, namesOf(globalVars(astFile)))

	var varsUsedAfterwards map[string]*ast.Ident

	var stmts []ast.Stmt

	switch typedParentNode := parentNode.(type) {
	case *ast.BlockStmt:
		var indexOfExtractedStmt int
		for i, stmt := range typedParentNode.List {
			if stmt == stmtsToExtract[0] {
				indexOfExtractedStmt = i
				break
			}
		}
		varsUsedAfterwards = varsWithTypesUsedIn(typedParentNode.List[indexOfExtractedStmt+len(stmtsToExtract):], varsDeclaredWithinStmtsToExtract)
		for _, node := range stmtsToExtract {
			stmts = append(stmts, node.(ast.Stmt))
		}
		if len(varsUsedAfterwards) == 0 {
			typedParentNode.List[indexOfExtractedStmt] = &ast.ExprStmt{X: callExprWith(extractedFuncName, params)}

		} else {
			typedParentNode.List[indexOfExtractedStmt] = &ast.AssignStmt{
				Lhs: []ast.Expr{ast.NewIdent(namesOf(varsUsedAfterwards)[0])},
				Tok: token.DEFINE,
				Rhs: []ast.Expr{callExprWith(extractedFuncName, params)},
			}

		}
		typedParentNode.List = append(typedParentNode.List[:indexOfExtractedStmt+1], typedParentNode.List[indexOfExtractedStmt+len(stmtsToExtract):]...)

	// TODO: Add cases for CommClause and CaseClause here

	default:
		panic(fmt.Sprintf("Type %v not supported yet", reflect.TypeOf(parentNode)))
	}

	astFile.Decls = append(astFile.Decls, multipleStmtFuncDeclWith(
		extractedFuncName,
		fieldsFrom(params),
		stmts,
		exprsFrom(varsUsedAfterwards),
	))
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
		returnType = &ast.FieldList{List: deduceTypes(definedVars)}
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
