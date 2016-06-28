// Copyright 2015 Peter Goetz
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"os/exec"

	"github.com/petergtz/goextract/util"
)

func ExtractFileToFile(inputFileName string, selection Selection, extractedFuncName string, outputFilename string, debugOutput bool) {
	fileSet, astFile := astFromFile(inputFileName)
	if debugOutput {
		createAstFileDump(inputFileName+".ast", fileSet, astFile)
	}
	doExtraction(fileSet, astFile, selection, extractedFuncName)
	util.WriteFileAsStringOrPanic(outputFilename, stringFrom(fileSet, astFile))
	err := exec.Command("gofmt", "-w", outputFilename).Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, util.ReadFileAsStringOrPanic(outputFilename))
		panic(err)
	}
}

func ExtractFileToString(inputFileName string, selection Selection, extractedFuncName string, debugOutput bool) string {
	fileSet, astFile := astFromFile(inputFileName)
	if debugOutput {
		createAstFileDump(inputFileName+".ast", fileSet, astFile)
	}
	doExtraction(fileSet, astFile, selection, extractedFuncName)
	return stringFrom(fileSet, astFile)
}

func ExtractStringToString(input string, selection Selection, extractedFuncName string) string {
	fileSet, astFile := astFromInput(input)
	doExtraction(fileSet, astFile, selection, extractedFuncName)
	return stringFrom(fileSet, astFile)
}

func doExtraction(fileSet *token.FileSet, astFile *ast.File, selection Selection, extractedFuncName string) {
	expression, parentNode := matchExpression(fileSet, astFile, selection)
	if expression != nil {
		extractExpressionAsFunc(astFile, fileSet, expression, parentNode, extractedFuncName)
	} else {
		stmts, parentNode := matchMultipleStmts(fileSet, astFile, selection)
		extractMultipleStatementsAsFunc(astFile, fileSet, stmts, parentNode, extractedFuncName)
	}
}
