package util

import "go/ast"

func MapStringStringRemoveKeys(m map[string]string, keys []string) {
	for _, key := range keys {
		delete(m, key)
	}
}

func MapStringAstIdentRemoveKeys(m map[string]*ast.Ident, keys []string) {
	for _, key := range keys {
		delete(m, key)
	}
}
