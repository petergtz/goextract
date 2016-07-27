[![Build Status](https://travis-ci.org/petergtz/goextract.svg?branch=master)](https://travis-ci.org/petergtz/goextract)
# goextract

A method extraction tool for the [Go](https://golang.org/) language.

This work is in very early development. The goal is to provide a refactoring tool that can [extract a method](http://refactoring.com/catalog/extractMethod.html) from Go source code.

See [test_data](https://github.com/petergtz/goextract/tree/master/test_data) to see what extractions are currently supported.

## Getting Started

### Getting It 
    go get github.com/petergtz/goextract

### Using It

Make sure `$GOPATH/bin` is in your `$PATH`.

Let's assume you have a file `main.go`,  with the content:

    1   package main
    2
    3   func g() {}
    4   func h() {}
    5   func i() {}
    6
    7   func f() {
    8       g()
    9       h()
    10      i()
    11  }

Then you can extract lines 9 and 10 into a function like this:

    goextract main.go --selection 9:1-11:1 --function MyExtractedFunc

The output will look like this:

package test_data

    1    func g() {}
    2    func h() {}
    3    func i() {}
    4
    5    func f() {
    6        g()
    7        MyExtractedFunc()
    8    }
    9
    10    func MyExtractedFunc() {
    11        h()
    12        i()
    13    }

goextract is quite smart in recognizing local variables or expression and will usually do the right thing during the extraction to make sure the logic of your code didn't change.

## Caveats

Please note that goextract doesn't handle comments correctly yet. If your code contains any kinds of comments anywhere, it's not recommended yet to use goextract.

## Using goextract in Your Editor

There's currently a [goextract extension](https://atom.io/packages/goextract) for the [Atom](https://atom.io/) editor.

Support for other editors is planned.