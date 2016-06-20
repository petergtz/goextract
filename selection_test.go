package main_test

import (
	. "github.com/petergtz/goextract"
	"github.com/petergtz/goextract/util"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Selection", func() {
	It("Shrinks correctly", func() {
		Expect(ShrinkToNonWhiteSpace(
			Selection{Begin: Position{4, 1}, End: Position{6, 1}},
			util.ReadFileAsStringOrPanic("test_data/shrink_selection"))).
			To(Equal(Selection{Begin: Position{5, 2}, End: Position{5, 8}}))
	})
})
