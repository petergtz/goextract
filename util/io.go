package util

import "io/ioutil"

func ReadFileAsStringOrPanic(filename string) string {
	content, err := ioutil.ReadFile(filename)
	PanicOnError(err)
	return string(content)
}
func WriteFileAsStringOrPanic(filename string, content string) {
	err := ioutil.WriteFile(filename, []byte(content), 0644)
	PanicOnError(err)
}
