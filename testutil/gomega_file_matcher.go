// Copyright 2016 Peter Goetz
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

package testutil

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"
)

type fileContentMatcher struct {
	actualFileContent []byte
	expectedFilePath  string
	actualFilePath    string
}

func HaveSameContentAs(filename string) *fileContentMatcher {
	return &fileContentMatcher{expectedFilePath: filename}
}

func (matcher *fileContentMatcher) Match(actual interface{}) (bool, error) {
	var ok bool
	matcher.actualFilePath, ok = actual.(string)
	if !ok {
		return false, fmt.Errorf("Matcher expects the actual file path as string. Instead got: %v", actual)
	}
	var err error
	matcher.actualFileContent, err = ioutil.ReadFile(matcher.actualFilePath)
	if err != nil {
		return false, fmt.Errorf("Actual file content could not be read from %v due to: %v", matcher.actualFilePath, err)
	}
	expectedFileContent, err := ioutil.ReadFile(matcher.expectedFilePath)
	if err != nil {
		return false, fmt.Errorf("Expected file content could not be read from %v due to: %v", matcher.expectedFilePath, err)
	}

	return string(matcher.actualFileContent) == string(expectedFileContent), nil
}

func (matcher *fileContentMatcher) FailureMessage(actual interface{}) string {
	out, err := exec.Command("colordiff", "-u", matcher.actualFilePath, matcher.expectedFilePath).Output()
	if err != nil && strings.Contains(err.Error(), "file not found") {
		out, err = exec.Command("diff", "-u", matcher.actualFilePath, matcher.expectedFilePath).Output()
	}
	exitError, ok := err.(*exec.ExitError)
	if !ok {
		panic(fmt.Errorf("Expected diff to exit with status 1, but got: %v", err))
	}
	if exitError.Error() != "exit status 1" {
		panic(fmt.Errorf("Expected diff to exit with status 1, but got: %v", err))
	}
	return fmt.Sprintf("Expected:\n\t%s to have same content as %s\n\nBut diff shows:\n%s",
		matcher.actualFilePath, matcher.expectedFilePath, out)
}

func (matcher *fileContentMatcher) NegatedFailureMessage(actual interface{}) string {
	return fmt.Sprintf("Expected:\n\t%s not to have same content as %s\n\nBut they both contain:\n%s",
		matcher.actualFilePath, matcher.expectedFilePath, string(matcher.actualFileContent))
}
