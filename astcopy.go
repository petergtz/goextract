package main

import (
	"fmt"
	"go/ast"
)

var (
	visitedNodes   map[ast.Node]ast.Node
	visitedObjects map[*ast.Object]*ast.Object
)

func copyExprSlice(slice []ast.Expr) []ast.Expr {
	if slice == nil {
		return nil
	}
	result := make([]ast.Expr, len(slice))
	for i, el := range slice {
		result[i] = copyNode(el).(ast.Expr)
	}
	return result
}

func copyStmtSlice(slice []ast.Stmt) []ast.Stmt {
	if slice == nil {
		return nil
	}
	result := make([]ast.Stmt, len(slice))
	for i, el := range slice {
		result[i] = copyNode(el).(ast.Stmt)
	}
	return result
}

func copyDeclSlice(slice []ast.Decl) []ast.Decl {
	if slice == nil {
		return nil
	}
	result := make([]ast.Decl, len(slice))
	for i, el := range slice {
		result[i] = copyNode(el).(ast.Decl)
	}
	return result
}

func copySpecSlice(slice []ast.Spec) []ast.Spec {
	if slice == nil {
		return nil
	}
	result := make([]ast.Spec, len(slice))
	for i, el := range slice {
		result[i] = copyNode(el).(ast.Spec)
	}
	return result
}

func copyCommentSlice(slice []*ast.Comment) []*ast.Comment {
	if slice == nil {
		return nil
	}
	result := make([]*ast.Comment, len(slice))
	for i, el := range slice {
		result[i] = copyNode(el).(*ast.Comment)
	}
	return result
}

func copyIdentSlice(slice []*ast.Ident) []*ast.Ident {
	if slice == nil {
		return nil
	}
	result := make([]*ast.Ident, len(slice))
	for i, el := range slice {
		result[i] = copyIdent(el)
	}
	return result
}

func copyFieldSlice(slice []*ast.Field) []*ast.Field {
	if slice == nil {
		return nil
	}
	result := make([]*ast.Field, len(slice))
	for i, el := range slice {
		result[i] = copyNode(el).(*ast.Field)
	}
	return result
}
func copyCommentGroupSlice(slice []*ast.CommentGroup) []*ast.CommentGroup {
	if slice == nil {
		return nil
	}
	result := make([]*ast.CommentGroup, len(slice))
	for i, el := range slice {
		result[i] = copyNode(el).(*ast.CommentGroup)
	}
	return result
}

func copyImportSpecSlice(slice []*ast.ImportSpec) []*ast.ImportSpec {
	if slice == nil {
		return nil
	}
	result := make([]*ast.ImportSpec, len(slice))
	for i, el := range slice {
		result[i] = copyNode(el).(*ast.ImportSpec)
	}
	return result
}

func copyScope(scope *ast.Scope) *ast.Scope {
	if scope == nil {
		return nil
	}
	return &ast.Scope{Objects: copyObjectsMap(scope.Objects), Outer: copyScope(scope.Outer)}
}

func copyObject(object *ast.Object) *ast.Object {
	if object == nil {
		return nil
	}
	if visitedObjects[object] != nil {
		return visitedObjects[object]
	}

	result := &ast.Object{
		Kind: object.Kind,
		Name: object.Name,
		// Decl: copyNode(object.Decl.(ast.Node)), // corresponding Field, XxxSpec, FuncDecl, LabeledStmt, AssignStmt, Scope; or nil
		Data: object.Data, // object-specific data; or nil
		Type: object.Type, // placeholder for type information; may be nil
	}
	visitedObjects[object] = result
	result.Decl = copyNode(object.Decl.(ast.Node))
	return result
}

func copyObjectsMap(m map[string]*ast.Object) map[string]*ast.Object {
	return nil
}

func copyFilesMap(m map[string]*ast.File) map[string]*ast.File {
	return nil
}

func CopyNode(node ast.Node) ast.Node {
	visitedNodes = make(map[ast.Node]ast.Node)
	visitedObjects = make(map[*ast.Object]*ast.Object)
	return copyNode(node)
}

func copyBlockStmt(n *ast.BlockStmt) *ast.BlockStmt {
	if n == nil {
		return nil
	}
	if visitedNodes[n] != nil {
		return visitedNodes[n].(*ast.BlockStmt)
	}

	result := &ast.BlockStmt{Lbrace: n.Lbrace, List: copyStmtSlice(n.List), Rbrace: n.Rbrace}
	visitedNodes[n] = result
	return result
}

func copyCommentGroup(n *ast.CommentGroup) *ast.CommentGroup {
	if n == nil {
		return nil
	}
	if visitedNodes[n] != nil {
		return visitedNodes[n].(*ast.CommentGroup)
	}
	result := &ast.CommentGroup{List: copyCommentSlice(n.List)}
	visitedNodes[n] = result
	return result

}

func copyFieldList(n *ast.FieldList) *ast.FieldList {
	if n == nil {
		return nil
	}
	if visitedNodes[n] != nil {
		return visitedNodes[n].(*ast.FieldList)
	}
	result := &ast.FieldList{Closing: n.Closing, List: copyFieldSlice(n.List), Opening: n.Opening}
	visitedNodes[n] = result
	return result

}

func copyBasicLit(n *ast.BasicLit) *ast.BasicLit {
	if n == nil {
		return nil
	}
	if visitedNodes[n] != nil {
		return visitedNodes[n].(*ast.BasicLit)
	}
	result := &ast.BasicLit{Kind: n.Kind, Value: n.Value, ValuePos: n.ValuePos}
	visitedNodes[n] = result
	return result

}

func copyIdent(ident *ast.Ident) *ast.Ident {
	if ident == nil {
		return nil
	}
	if visitedNodes[ident] != nil {
		return visitedNodes[ident].(*ast.Ident)
	}
	result := &ast.Ident{
		Name:    ident.Name,
		NamePos: ident.NamePos,
		Obj:     copyObject(ident.Obj),
	}
	visitedNodes[ident] = result
	return result

}

func copyNode(node ast.Node) ast.Node {
	if node == nil {
		return nil
	}
	// if visitedNodes[node] != nil {
	// 	return visitedNodes[node]
	// }

	var result ast.Node
	switch n := node.(type) {
	case *ast.ArrayType:
		result = &ast.ArrayType{
			Lbrack: n.Lbrack,
			Len:    typeAssertToExpr(copyNode(n.Len)),
			Elt:    typeAssertToExpr(copyNode(n.Elt)),
		}
	case *ast.AssignStmt:
		result = &ast.AssignStmt{
			Lhs:    copyExprSlice(n.Lhs),
			Rhs:    copyExprSlice(n.Rhs),
			Tok:    n.Tok,
			TokPos: n.TokPos,
		}
	case *ast.BadDecl:
		result = &ast.BadDecl{From: n.From, To: n.To}
	case *ast.BadExpr:
		result = &ast.BadExpr{From: n.From, To: n.To}
	case *ast.BadStmt:
		result = &ast.BadStmt{From: n.From, To: n.To}
	case *ast.BasicLit:
		result = &ast.BasicLit{Kind: n.Kind, Value: n.Value, ValuePos: n.ValuePos}
	case *ast.BinaryExpr:
		result = &ast.BinaryExpr{Op: n.Op, OpPos: n.OpPos, X: copyNode(n.X).(ast.Expr), Y: copyNode(n.Y).(ast.Expr)}
	case *ast.BlockStmt:
		result = &ast.BlockStmt{Lbrace: n.Lbrace, List: copyStmtSlice(n.List), Rbrace: n.Rbrace}
	case *ast.BranchStmt:
		result = &ast.BranchStmt{Label: copyIdent(n.Label), Tok: n.Tok, TokPos: n.TokPos}
	case *ast.CallExpr:
		result = &ast.CallExpr{Args: copyExprSlice(n.Args), Ellipsis: n.Ellipsis, Fun: copyNode(n.Fun).(ast.Expr), Lparen: n.Lparen, Rparen: n.Rparen}
	case *ast.CaseClause:
		result = &ast.CaseClause{Body: copyStmtSlice(n.Body), Case: n.Case, Colon: n.Colon, List: copyExprSlice(n.List)}
	case *ast.ChanType:
		result = &ast.ChanType{Arrow: n.Arrow, Begin: n.Begin, Dir: n.Dir, Value: copyNode(n.Value).(ast.Expr)}
	case *ast.CommClause:
		result = &ast.CommClause{Body: copyStmtSlice(n.Body), Case: n.Case, Colon: n.Colon, Comm: copyNode(n.Comm).(ast.Stmt)}
	case *ast.Comment:
		result = &ast.Comment{Slash: n.Slash, Text: n.Text}
	case *ast.CommentGroup:
		result = &ast.CommentGroup{List: copyCommentSlice(n.List)}
	case *ast.CompositeLit:
		result = &ast.CompositeLit{Elts: copyExprSlice(n.Elts), Lbrace: n.Lbrace, Rbrace: n.Rbrace, Type: copyNode(n.Type).(ast.Expr)}
	case *ast.DeclStmt:
		result = &ast.DeclStmt{Decl: copyNode(n.Decl).(ast.Decl)}
	case *ast.DeferStmt:
		result = &ast.DeferStmt{Call: copyNode(n.Call).(*ast.CallExpr), Defer: n.Defer}
	case *ast.Ellipsis:
		result = &ast.Ellipsis{Ellipsis: n.Ellipsis, Elt: copyNode(n.Elt).(ast.Expr)}
	case *ast.EmptyStmt:
		result = &ast.EmptyStmt{Implicit: n.Implicit, Semicolon: n.Semicolon}
	case *ast.ExprStmt:
		result = &ast.ExprStmt{X: copyNode(n.X).(ast.Expr)}
	case *ast.Field:
		result = &ast.Field{Comment: copyCommentGroup(n.Comment), Doc: copyCommentGroup(n.Doc), Names: copyIdentSlice(n.Names), Tag: copyBasicLit(n.Tag), Type: copyNode(n.Type).(ast.Expr)}
	case *ast.FieldList:
		result = copyFieldList(n)
	case *ast.File:
		result = &ast.File{
			Comments:   copyCommentGroupSlice(n.Comments),
			Decls:      copyDeclSlice(n.Decls),
			Doc:        copyCommentGroup(n.Doc),
			Imports:    copyImportSpecSlice(n.Imports),
			Name:       copyIdent(n.Name),
			Package:    n.Package,
			Scope:      copyScope(n.Scope),
			Unresolved: copyIdentSlice(n.Unresolved),
		}
	case *ast.ForStmt:
		result = &ast.ForStmt{Body: copyBlockStmt(n.Body), Cond: copyNode(n.Cond).(ast.Expr), For: n.For, Init: copyNode(n.Init).(ast.Stmt), Post: copyNode(n.Post).(ast.Stmt)}
	case *ast.FuncDecl:
		result = &ast.FuncDecl{Body: copyBlockStmt(n.Body), Doc: copyCommentGroup(n.Doc), Name: copyNode(n.Name).(*ast.Ident), Recv: copyFieldList(n.Recv), Type: copyNode(n.Type).(*ast.FuncType)}
	case *ast.FuncLit:
		result = &ast.FuncLit{Body: copyBlockStmt(n.Body), Type: copyNode(n.Type).(*ast.FuncType)}
	case *ast.FuncType:
		result = &ast.FuncType{Func: n.Func, Params: copyFieldList(n.Params), Results: copyFieldList(n.Results)}
	case *ast.GenDecl:
		result = &ast.GenDecl{Doc: copyCommentGroup(n.Doc), Lparen: n.Lparen, Rparen: n.Rparen, Specs: copySpecSlice(n.Specs), Tok: n.Tok, TokPos: n.TokPos}
	case *ast.GoStmt:
		result = &ast.GoStmt{Call: copyNode(n.Call).(*ast.CallExpr), Go: n.Go}
	case *ast.Ident:
		result = &ast.Ident{Name: n.Name, NamePos: n.NamePos, Obj: copyObject(n.Obj)}
	case *ast.IfStmt:
		result = &ast.IfStmt{Body: copyBlockStmt(n.Body), Cond: typeAssertToExpr(copyNode(n.Cond)), Else: typeAssertToStmt(copyNode(n.Else)), If: n.If, Init: typeAssertToStmt(copyNode(n.Init))}
	case *ast.ImportSpec:
		result = &ast.ImportSpec{Comment: copyCommentGroup(n.Comment), Doc: copyCommentGroup(n.Doc), EndPos: n.EndPos, Name: copyNode(n.Name).(*ast.Ident), Path: copyBasicLit(n.Path)}
	case *ast.IncDecStmt:
		result = &ast.IncDecStmt{Tok: n.Tok, TokPos: n.TokPos, X: copyNode(n.X).(ast.Expr)}
	case *ast.IndexExpr:
		result = &ast.IndexExpr{Index: copyNode(n.Index).(ast.Expr), Lbrack: n.Lbrack, Rbrack: n.Rbrack, X: copyNode(n.X).(ast.Expr)}
	case *ast.InterfaceType:
		result = &ast.InterfaceType{Incomplete: n.Incomplete, Interface: n.Interface, Methods: copyFieldList(n.Methods)}
	case *ast.KeyValueExpr:
		result = &ast.KeyValueExpr{Colon: n.Colon, Key: copyNode(n.Key).(ast.Expr), Value: copyNode(n.Value).(ast.Expr)}
	case *ast.LabeledStmt:
		result = &ast.LabeledStmt{Colon: n.Colon, Label: copyIdent(n.Label), Stmt: copyNode(n.Stmt).(ast.Stmt)}
	case *ast.MapType:
		result = &ast.MapType{Key: copyNode(n.Key).(ast.Expr), Map: n.Map, Value: copyNode(n.Value).(ast.Expr)}
	case *ast.Package:
		result = &ast.Package{Files: copyFilesMap(n.Files), Imports: copyObjectsMap(n.Imports), Name: n.Name, Scope: copyScope(n.Scope)}
	case *ast.ParenExpr:
		result = &ast.ParenExpr{Lparen: n.Lparen, Rparen: n.Rparen, X: copyNode(n.X).(ast.Expr)}
	case *ast.RangeStmt:
		result = &ast.RangeStmt{Body: copyBlockStmt(n.Body), For: n.For, Key: copyNode(n.Key).(ast.Expr), Tok: n.Tok, TokPos: n.TokPos, Value: copyNode(n.Value).(ast.Expr), X: copyNode(n.X).(ast.Expr)}
	case *ast.ReturnStmt:
		result = &ast.ReturnStmt{Results: copyExprSlice(n.Results), Return: n.Return}
	case *ast.SelectStmt:
		result = &ast.SelectStmt{Body: copyBlockStmt(n.Body), Select: n.Select}
	case *ast.SelectorExpr:
		result = &ast.SelectorExpr{Sel: copyIdent(n.Sel), X: copyNode(n.X).(ast.Expr)}
	case *ast.SendStmt:
		result = &ast.SendStmt{Arrow: n.Arrow, Chan: copyNode(n.Chan).(ast.Expr), Value: copyNode(n.Value).(ast.Expr)}
	case *ast.SliceExpr:
		result = &ast.SliceExpr{High: copyNode(n.High).(ast.Expr), Lbrack: n.Lbrack, Low: copyNode(n.Low).(ast.Expr), Max: copyNode(n.Max).(ast.Expr), Rbrack: n.Rbrack, Slice3: n.Slice3, X: copyNode(n.X).(ast.Expr)}
	case *ast.StarExpr:
		result = &ast.StarExpr{Star: n.Star, X: copyNode(n.X).(ast.Expr)}
	case *ast.StructType:
		result = &ast.StructType{
			Struct:     n.Struct,
			Fields:     copyFieldList(n.Fields),
			Incomplete: n.Incomplete,
		}
	case *ast.SwitchStmt:
		result = &ast.SwitchStmt{Body: copyBlockStmt(n.Body), Init: copyNode(n.Init).(ast.Stmt), Switch: n.Switch, Tag: copyNode(n.Tag).(ast.Expr)}
	case *ast.TypeAssertExpr:
		var ty ast.Expr
		if t := copyNode(n.Type); t != nil {
			ty = t.(ast.Expr)
		}
		var x ast.Expr
		if tx := copyNode(n.X); tx != nil {
			x = tx.(ast.Expr)
		}
		result = &ast.TypeAssertExpr{Lparen: n.Lparen, Rparen: n.Rparen, Type: ty, X: x}
	case *ast.TypeSpec:
		result = &ast.TypeSpec{Comment: copyCommentGroup(n.Comment), Doc: copyCommentGroup(n.Doc), Name: copyNode(n.Name).(*ast.Ident), Type: copyNode(n.Type).(ast.Expr)}
	case *ast.TypeSwitchStmt:
		result = &ast.TypeSwitchStmt{Assign: typeAssertToStmt(copyNode(n.Assign)), Body: copyBlockStmt(n.Body), Init: typeAssertToStmt(copyNode(n.Init)), Switch: n.Switch}
	case *ast.UnaryExpr:
		result = &ast.UnaryExpr{Op: n.Op, OpPos: n.OpPos, X: copyNode(n.X).(ast.Expr)}
	case *ast.ValueSpec:
		result = &ast.ValueSpec{Comment: copyCommentGroup(n.Comment), Doc: copyCommentGroup(n.Doc), Names: copyIdentSlice(n.Names), Type: typeAssertToExpr(copyNode(n.Type)), Values: copyExprSlice(n.Values)}
	default:
		panic(fmt.Sprintf("Node copy for %T not implemented yet", n))
	}

	if visitedNodes[node] == nil {
		visitedNodes[node] = result
	}
	return result
}

func typeAssertToExpr(node ast.Node) ast.Expr {
	if node == nil {
		return nil
	}
	return node.(ast.Expr)
}

func typeAssertToStmt(node ast.Node) ast.Stmt {
	if node == nil {
		return nil
	}
	return node.(ast.Stmt)
}
