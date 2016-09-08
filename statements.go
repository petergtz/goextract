package main

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/token"
	"go/types"
	"os"
	"reflect"

	"strings"

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

func matchMultipleStmts(fileSet *token.FileSet, astFile *ast.File, selection Selection) ([]ast.Node, ast.Node) {
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
	return v.context.nodesToExtract, v.context.posParent
}

func extractMultipleStatementsAsFunc(
	astFile *ast.File,
	fileSet *token.FileSet,
	stmtsToExtract []ast.Node,
	parentNode ast.Node,
	extractedFuncName string) {

	pkg, _ := typesPackage(astFile, fileSet)

	params := varIdentsUsedIn(stmtsToExtract)
	varsDeclaredWithinStmtsToExtract := varIdentsDeclaredWithin(stmtsToExtract)
	util.MapStringAstIdentRemoveKeys(params, namesOf(varsDeclaredWithinStmtsToExtract))
	util.MapStringAstIdentRemoveKeys(params, namesOf(globalVarIdents(astFile)))

	allStmts := stmtsFromBlockStmt(parentNode)
	indexOfExtractedStmt := indexOf(stmtsToExtract[0].(ast.Stmt), *allStmts)
	varsUsedAfterwards := overlappingVarsIdentsUsedIn((*allStmts)[indexOfExtractedStmt+len(stmtsToExtract):], varsDeclaredWithinStmtsToExtract)

	newStmt := funcCallStmt(varsUsedAfterwards, extractedFuncName, params, (*allStmts)[indexOfExtractedStmt].Pos())
	RecalcPoses(newStmt, (*allStmts)[indexOfExtractedStmt].Pos(), nil, 0)

	replaceStmtsWithFuncCallStmt(newStmt,
		allStmts,
		indexOfExtractedStmt, len(stmtsToExtract))

	removedComments := removeComments(astFile, (stmtsToExtract)[0].Pos(), (stmtsToExtract)[len(stmtsToExtract)-1].End())

	shiftPosesAfterPos(astFile, newStmt, (stmtsToExtract)[len(stmtsToExtract)-1].End(), newStmt.End()-stmtsToExtract[len(stmtsToExtract)-1].End())

	multipleStmtFuncDecl := CopyNode(multipleStmtFuncDeclWith(
		extractedFuncName,
		fieldsFrom(params, pkg),
		stmtsFromNodes(stmtsToExtract),
		exprsFrom(varsUsedAfterwards),
		pkg,
	)).(*ast.FuncDecl)
	var moveOffset token.Pos
	RecalcPoses(multipleStmtFuncDecl, astFile.End()+2, &moveOffset, 0)
	astFile.Comments = append(astFile.Comments, removedComments...)
	astFile.Decls = append(astFile.Decls, multipleStmtFuncDecl)

	moveComments(astFile, moveOffset, (stmtsToExtract)[0].Pos(), (stmtsToExtract)[len(stmtsToExtract)-1].End())

	areaRemoved := areaRemoved(fileSet, (stmtsToExtract)[0].Pos(), (stmtsToExtract)[len(stmtsToExtract)-1].End())
	lineLengths := lineLengthsFrom(fileSet)
	lineNum, numLinesToCut, newLineLength := replacementModifications(
		fileSet, (stmtsToExtract)[0].Pos(), (stmtsToExtract)[len(stmtsToExtract)-1].End(), newStmt.End(), lineLengths, areaRemoved)
	lineLengths = append(
		lineLengths[:lineNum+1],
		lineLengths[lineNum+1+numLinesToCut:]...)
	lineLengths[lineNum] = newLineLength
	lineLengths = append(lineLengths, areaToBeAppendedForStmts(multipleStmtFuncDecl, areaRemoved, exprsFrom(varsUsedAfterwards))...)

	newFileSet := token.NewFileSet()
	newFileSet.AddFile(fileSet.File(1).Name(), 1, int(astFile.End()))
	success := newFileSet.File(1).SetLines(ConvertLineLengthsToLineOffsets(lineLengths))
	if !success {
		panic("Could not SetLines on File.")
	}
	*fileSet = *newFileSet
}

func typesPackage(astFile *ast.File, fileSet *token.FileSet) (*types.Package, *types.Info) {
	conf := types.Config{Importer: importer.Default()}
	info := types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Uses:  make(map[*ast.Ident]types.Object),
		Defs:  make(map[*ast.Ident]types.Object),
	}
	pkg, err := conf.Check("some/path", fileSet, []*ast.File{astFile}, &info)
	fmt.Fprintln(os.Stderr, err)
	// util.PanicOnError(err)
	return pkg, &info
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

func replaceStmtsWithFuncCallStmt(funcCallStmt ast.Stmt, allStmts *[]ast.Stmt, indexOfExtractedStmt int, count int) {
	(*allStmts)[indexOfExtractedStmt] = funcCallStmt
	(*allStmts) = append((*allStmts)[:indexOfExtractedStmt+1], (*allStmts)[indexOfExtractedStmt+count:]...)
}

func funcCallStmt(varsUsedAfterwards map[string]*ast.Ident, extractedFuncName string, params map[string]*ast.Ident, pos token.Pos) (result ast.Stmt) {
	if len(varsUsedAfterwards) == 0 {
		result = CopyNode(&ast.ExprStmt{X: callExprWith(extractedFuncName, params)}).(ast.Stmt)
	} else {
		result = CopyNode(&ast.AssignStmt{
			Lhs: []ast.Expr{ast.NewIdent(namesOf(varsUsedAfterwards)[0])},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{callExprWith(extractedFuncName, params)},
		}).(ast.Stmt)
	}
	return
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
	definedVars []ast.Expr,
	pkg *types.Package) *ast.FuncDecl {

	allStmts := make([]ast.Stmt, len(stmts), len(stmts)+1)
	copy(allStmts, stmts)
	var returnType *ast.FieldList
	definedVarsCopy := copyExprSlice(definedVars)
	fieldListCopy := fieldListFromIdents(definedVarsCopy, pkg)
	for _, t := range fieldListCopy {
		resetPoses(t)
	}
	for _, t := range definedVarsCopy {
		resetPoses(t)
	}
	if len(definedVarsCopy) != 0 {
		allStmts = append(allStmts, &ast.ReturnStmt{Results: definedVarsCopy})

		returnType = &ast.FieldList{List: fieldListCopy}
	}
	fieldsCopy := copyFieldSlice(fields)
	for _, t := range fieldsCopy {
		resetPoses(t)
	}
	return &ast.FuncDecl{
		Name: ast.NewIdent(extractedFuncName),
		Type: &ast.FuncType{
			Params:  &ast.FieldList{List: fieldsCopy},
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

func fieldListFromIdents(idents []ast.Expr, pkg *types.Package) []*ast.Field {
	var fieldList []*ast.Field
	for _, ident := range idents {
		fieldList = append(fieldList, &ast.Field{Type: deduceWithTypeLib(ident.(*ast.Ident), pkg)})
	}
	return fieldList
}

func deduceWithTypeLib(ident *ast.Ident, pkg *types.Package) *ast.Ident {
	_, object := pkg.Scope().Innermost(ident.Pos()).LookupParent(ident.Name, 0)
	if object == nil {
		// panic(fmt.Sprint("Cannot look up ident with name ", ident.Name, " and pos ", ident.Pos()))
		fmt.Fprintln(os.Stderr, "Cannot look up ident with name ", ident.Name, " and pos ", ident.Pos())
		return ast.NewIdent("TypeNotFound")
	}
	typeString := object.Type().String()
	for _, imp := range pkg.Imports() {
		if strings.Contains(typeString, imp.Path()) {
			typeString = strings.Replace(typeString, imp.Path(), imp.Name(), 1)
			break
		}
	}
	// seems necessary for structs declared in this package:
	typeString = strings.Replace(typeString, pkg.Path()+".", "", 1)

	return ast.NewIdent(typeString)
}
