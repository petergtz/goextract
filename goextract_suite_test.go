package main_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestGoextract(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Goextract Suite")
}
