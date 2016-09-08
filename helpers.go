package main

import (
	"go/ast"
	"go/token"
	"go/types"
	"sort"
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
			case *ast.ValueSpec:
				for _, name := range typedNode.Names {
					result[name.Name] = name
				}

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
		result[i] = CopyNode(idents[key]).(ast.Expr)
	}
	return result
}

func callExprWith(funcName string, params map[string]*ast.Ident) *ast.CallExpr {
	fun := &ast.Ident{Name: funcName /*, NamePos: pos*/}
	args := exprsFrom(params)
	// currentPos := fun.End() + 2
	// for _, arg := range args {
	// 	adjustPoses(arg, currentPos)
	// 	currentPos = arg.End() + 1
	// }
	return &ast.CallExpr{
		Fun: fun,
		// Lparen: fun.End() + 1,
		Args: args,
		// Rparen: currentPos,
	}
}

func adjustPoses(node ast.Node, pos token.Pos) {

}

func fieldsFrom(params map[string]*ast.Ident, pkg *types.Package) []*ast.Field {
	result := make([]*ast.Field, len(params))
	for i, key := range sortedKeysFrom(params) {
		result[i] = &ast.Field{
			Names: []*ast.Ident{ast.NewIdent(key)},
			Type:  deduceWithTypeLib(params[key], pkg),
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
