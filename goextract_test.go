package main_test

import (
	. "github.com/petergtz/goextract"
	"github.com/petergtz/goextract/util"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Goextract", func() {
	It("Can extract a single declaration", func() {
		output := ExtractFileToString("single_declaration.go.input", Selection{Position{7, 5}, Position{7, 6}}, "MyExtractedFunc")
		Expect(output).To(Equal(util.ReadFileAsStringOrPanic("single_declaration.go.output")))
	})
})
