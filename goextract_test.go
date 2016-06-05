package main_test

import (
	"io/ioutil"
	"strconv"
	"strings"

	. "github.com/petergtz/goextract"
	"github.com/petergtz/goextract/util"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Goextract", func() {
	fileInfos, err := ioutil.ReadDir("test_data")
	util.PanicOnError(err)
	for _, fileInfo := range fileInfos {
		filename := fileInfo.Name()
		if !strings.HasSuffix(filename, ".input") {
			continue
		}
		prefix := strings.TrimSuffix(filename, ".input")

		It("Can extract a "+strings.Replace(prefix, "_", " ", -1), func() {
			selection, extractedFuncName := selectionFrom("test_data/" + prefix + ".extract")

			output := ExtractFileToString("test_data/"+filename, selection, extractedFuncName)

			Expect(output).To(Equal(util.ReadFileAsStringOrPanic("test_data/" + prefix + ".output")))
		})

	}
})

func selectionFrom(filename string) (Selection, string) {
	parts := strings.Split(strings.TrimRight(util.ReadFileAsStringOrPanic(filename), "\n"), " ")
	Expect(parts).To(HaveLen(5))
	return Selection{
			Position{toInt(parts[0]), toInt(parts[1])},
			Position{toInt(parts[2]), toInt(parts[3])},
		},
		parts[4]
}

func toInt(s string) int {
	i, err := strconv.Atoi(s)
	util.PanicOnError(err)
	return i
}
