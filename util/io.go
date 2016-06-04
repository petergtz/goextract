package util

import "io/ioutil"

func ReadFileAsStringOrPanic(filename string) string {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	return string(content)
}
