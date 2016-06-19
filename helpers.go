package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"sort"
	"strings"
)

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
			switch typedNode := node.(type) {

			case *ast.AssignStmt:
				if typedNode.Tok.String() == ":=" {
					for i := range typedNode.Lhs {
						result[typedNode.Lhs[i].(*ast.Ident).Name] = typedNode.Lhs[i].(*ast.Ident)
					}
				}

			case *ast.RangeStmt:
				if typedNode.Tok.String() == ":=" {
					result[typedNode.Key.(*ast.Ident).Name] = typedNode.Key.(*ast.Ident)
					if typedNode.Value != nil {
						result[typedNode.Value.(*ast.Ident).Name] = typedNode.Value.(*ast.Ident)
					}
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
				result[ident.Name] = ident
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
	result := make([]string, len(idents))
	for i, key := range sortedKeysFrom(idents) {
		result[i] = key
	}
	return result
}

func exprsFrom(idents map[string]*ast.Ident) []ast.Expr {
	result := make([]ast.Expr, len(idents))
	for i, key := range sortedKeysFrom(idents) {
		result[i] = idents[key]
	}
	return result
}

func callExprWith(funcName string, params map[string]*ast.Ident) *ast.CallExpr {
	return &ast.CallExpr{
		Fun:  ast.NewIdent(funcName),
		Args: exprsFrom(params),
	}
}

func fieldsFrom(params map[string]*ast.Ident) []*ast.Field {
	result := make([]*ast.Field, len(params))
	for i, key := range sortedKeysFrom(params) {
		result[i] = &ast.Field{
			Names: []*ast.Ident{ast.NewIdent(key)},
			Type:  deduceTypeExprForVarIdent(params[key]),
		}
	}
	return result
}

func sortedKeysFrom(m map[string]*ast.Ident) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func deduceTypeExprsForExpr(expr ast.Expr) []ast.Expr {
	switch typedExpr := expr.(type) {
	case *ast.Ident:
		// TODO why do we even get type-like idents in here? Shouldn't this be already be handled one level higher?
		if typedExpr.Obj == nil {
			return []ast.Expr{typedExpr}
		}
		switch typedExpr.Obj.Kind {
		case ast.Var:
			return []ast.Expr{deduceTypeExprForVarIdent(typedExpr)}
		case ast.Typ:
			return []ast.Expr{typedExpr}
		default:
			panic("Unexpected ident obj kind")
		}
	case *ast.Ellipsis:
		return []ast.Expr{typedExpr.Elt.(*ast.Ident)}
	case *ast.BasicLit:
		return []ast.Expr{ast.NewIdent(strings.ToLower(typedExpr.Kind.String()))}
	case *ast.FuncLit:
		if typedExpr.Type.Results == nil {
			return nil
		}
		var result []ast.Expr
		for _, res := range typedExpr.Type.Results.List {
			result = append(result, res.Type.(*ast.Ident))
		}
		return result
	case *ast.CompositeLit:
		return deduceTypeExprsForExpr(typedExpr.Type)
	case *ast.ParenExpr:
		return deduceTypeExprsForExpr(typedExpr.X)
	case *ast.SelectorExpr:
		return deduceTypeExprsForExpr(typedExpr.Sel)
	case *ast.IndexExpr:
		return deduceTypeExprsForExpr(typedExpr.X)
	case *ast.SliceExpr:
		return deduceTypeExprsForExpr(typedExpr.X)
	case *ast.TypeAssertExpr:
		if typedExpr.Type == nil {
			return []ast.Expr{ast.NewIdent("TypeSwitch")}
		} else {
			return []ast.Expr{typedExpr.Type.(*ast.Ident)}
		}
	case *ast.CallExpr:
		if typedExpr.Fun.(*ast.Ident).Obj == nil {
			return []ast.Expr{ast.NewIdent("UnresolvedIdent_" + typedExpr.Fun.(*ast.Ident).Name)}
		}
		if typedExpr.Fun.(*ast.Ident).Obj.Decl.(*ast.FuncDecl).Type.Results == nil {
			return nil
		}
		var result []ast.Expr
		for _, res := range typedExpr.Fun.(*ast.Ident).Obj.Decl.(*ast.FuncDecl).Type.Results.List {
			result = append(result, res.Type.(*ast.Ident))
		}
		return result
	case *ast.StarExpr:
		panic(fmt.Sprintf("Type deduction for %T not implemented yet", expr))
	case *ast.UnaryExpr:
		switch typedExpr.Op {
		case token.RANGE:
			return []ast.Expr{ast.NewIdent("int"), deduceTypeExprsForExpr(typedExpr.X)[0]}
		case token.AND:
			return []ast.Expr{&ast.StarExpr{X: deduceTypeExprsForExpr(typedExpr.X)[0]}}
		default:
			panic(fmt.Sprintf("UnaryExpr not implemented with Op \"%v\" yet", typedExpr.Op))
		}
	case *ast.BinaryExpr:
		return deduceTypeExprsForExpr(typedExpr.X)
	case *ast.KeyValueExpr:
		return []ast.Expr{typedExpr.Value.(*ast.Ident)}
	case *ast.ArrayType:
		return []ast.Expr{typedExpr.Elt.(*ast.Ident)}
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

// Note: a type can also be a *SelectorExpr e.g., therefore return type here
// cannot simply be *ast.Ident
func deduceTypeExprForVarIdent(ident *ast.Ident) ast.Expr {
	if ident.Obj.Kind != ast.Var {
		panic(fmt.Sprintf("Expected ObjKind \"var\" for ident, but got \"%v\"", ident.Obj.Kind))
	}
	switch typedDecl := ident.Obj.Decl.(type) {
	case *ast.AssignStmt:
		for i, lhs := range typedDecl.Lhs {
			if lhs.(*ast.Ident).Obj == ident.Obj {
				if len(typedDecl.Rhs) == 0 {
					panic("Unexpected empty Rhs")
				}
				if len(typedDecl.Rhs) == 1 {
					// this is the function call with multiple return values
					return deduceTypeExprsForExpr(typedDecl.Rhs[0])[i]
				} else {
					return deduceTypeExprsForExpr(typedDecl.Rhs[i])[0]
				}
			}
		}
		panic("Unexpected: no result in AssignStmt")
	case *ast.ValueSpec:
		for i, name := range typedDecl.Names {
			if name.Obj == ident.Obj {
				if typedDecl.Type != nil {
					return typedDecl.Type
				} else {
					if len(typedDecl.Values) == 0 {
						panic("Unexpected empty value")
					}
					if len(typedDecl.Values) == 1 {
						return deduceTypeExprsForExpr(typedDecl.Values[0])[i]
					} else {
						return deduceTypeExprsForExpr(typedDecl.Values[i])[0]
					}

				}
			}
		}
		panic("Unexpected: no result in ValueSpec")
	case *ast.Field:
		for _, name := range typedDecl.Names {
			if name.Obj == ident.Obj {
				return typedDecl.Type
			}
		}
		panic("Unexpected: no result in Field")

	default:
		panic(fmt.Sprintf("Unexpected decl type %T", typedDecl))
	}
}

func deduceTypeExprsForVarIdents(varIdents []*ast.Ident) (typeIdents []ast.Expr) {
	for _, ident := range varIdents {
		typeIdents = append(typeIdents, deduceTypeExprForVarIdent(ident))
	}
	return
}
