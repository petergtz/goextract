package util

import "strconv"

func ToInt(s string) int {
	i, err := strconv.Atoi(s)
	PanicOnError(err)
	return i
}
