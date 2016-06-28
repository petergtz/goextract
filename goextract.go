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
	visitor := &astNodeVisitorForExpressions{parentNode: nil, context: &expressionVisitorContext{fset: fileSet, selection: selection}}
	ast.Walk(visitor, astFile)
	if visitor.context.exprToExtract != nil {
		extractExpression(astFile, fileSet, visitor.context.exprToExtract, visitor.context.parent, extractedFuncName)
	} else {
		v := &astNodeVisitorForMultipleStatements{parentNode: nil, context: &multipleStatementVisitorContext{fset: fileSet, selection: selection}}
		ast.Walk(v, astFile)
		if v.context.posParent != v.context.endParent {
			panic(fmt.Sprintf("Selection is not valid. posParent: %v; endParent: %v",
				v.context.posParent, v.context.endParent))
		}
		if v.context.posParent == nil {
			panic(fmt.Sprintf("Selection is not valid. posParent: %v; endParent: %v",
				v.context.posParent, v.context.endParent))
		}
		extractMultipleStatements(astFile, fileSet, v.context.nodesToExtract, v.context.posParent, extractedFuncName)
	}
}
