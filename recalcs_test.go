package main_test

import (
	"go/ast"
	"go/parser"

	"go/token"

	"github.com/davecgh/go-spew/spew"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/petergtz/goextract"
)

var _ = Describe("Recalcs", func() {
	XIt("bla", func() {
		fileSet := token.NewFileSet()
		expectedAstFile, err := parser.ParseFile(fileSet, "test_data/trivial", nil, 0)
		Expect(err).NotTo(HaveOccurred())
		actualAstFile := &ast.File{
			Name: ast.NewIdent("test_data"),
			Decls: []ast.Decl{
				&ast.GenDecl{
					Tok: token.VAR,
					Specs: []ast.Spec{
						&ast.ValueSpec{
							Names: []*ast.Ident{ast.NewIdent("i")},
							Type:  ast.NewIdent("int"),
						},
					},
				},
			},
		}
		RecalcPoses(actualAstFile, 3, nil, 0)
		spew.Config.ContinueOnMethod = true
		Expect(spew.Sdump(actualAstFile)).To(Equal(spew.Sdump(expectedAstFile)))

	})

	It("Can ConvertLineOffsetsToLineLengths", func() {
		lineLengths := ConvertLineOffsetsToLineLengths([]int{0, 18, 19, 25, 38, 68, 72}, 105)
		Expect(lineLengths).To(ConsistOf([]int{17, 0, 5, 12, 29, 3, 33}))
	})

	It("Can ConvertLineLengthsToLineOffsets", func() {
		offsets := ConvertLineLengthsToLineOffsets([]int{17, 0, 5, 12, 29, 3, 33})
		Expect(offsets).To(ConsistOf([]int{0, 18, 19, 25, 38, 68, 72}))
	})
})
