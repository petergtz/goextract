package main_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	. "github.com/petergtz/goextract"
	. "github.com/petergtz/goextract/testutil"
	"github.com/petergtz/goextract/util"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// This is a workaround for the current way of creating test cases.
// A better way could be to actually move the test cases back into this file
// instead of looping automatically through all test files.
var focusedTests = map[string]bool{}

var pendingTests = map[string]bool{
	"multiple_simple_statements_taking_parameters_used_afterwards":                                        true,
	"multiple_simple_statements_taking_parameters_declared_beforehand_changed_within_and_used_afterwards": true,
	"multi_value_declaration_through_callexpr_within_range_loop":                                          true,
}

var _ = Describe("Goextract", func() {
	fileInfos, err := ioutil.ReadDir("test_data")
	util.PanicOnError(err)
	for _, fileInfo := range fileInfos {
		filename := fileInfo.Name()
		if !strings.HasSuffix(filename, ".go.input") {
			continue
		}
		prefix := strings.TrimSuffix(filename, ".go.input")

		it := It
		if pendingTests[filepath.Base(prefix)] {
			// TODO ideally this would use PIt to do better reporting, but it actually
			// has a differnt method signature.
			continue
		}
		if focusedTests[filepath.Base(prefix)] {
			it = FIt
		}

		it("Can extract a "+strings.Replace(prefix, "_", " ", -1), func() {
			selection, extractedFuncName := extractionDataFrom(filepath.Join("test_data", prefix) + ".go.extract")

			tmpfile, err := ioutil.TempFile("", "goextract")
			util.PanicOnError(err)
			defer os.Remove(tmpfile.Name())

			ExtractFileToFile(filepath.Join("test_data", filename), selection, extractedFuncName, tmpfile.Name())

			Expect(tmpfile.Name()).To(HaveSameContentAs(filepath.Join("test_data", prefix) + ".go.output"))
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
