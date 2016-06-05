package main_test

import (
	"io/ioutil"
	"path/filepath"
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
			selection, extractedFuncName := extractionDataFrom(filepath.Join("test_data", prefix) + ".extract")

			actualOutput := ExtractFileToString(filepath.Join("test_data", filename), selection, extractedFuncName)

			Expect(actualOutput).To(Equal(expectedOutputFor(prefix)))
		})

	}
})

func extractionDataFrom(filename string) (Selection, string) {
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

func expectedOutputFor(prefix string) string {
	return util.ReadFileAsStringOrPanic(filepath.Join("test_data", prefix) + ".output")
}
