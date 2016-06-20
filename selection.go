package main

import (
	"strings"

	"github.com/petergtz/goextract/util"
)

type Selection struct {
	Begin, End Position
}

type Position struct {
	Line, Column int
}

func ShrinkToNonWhiteSpace(selection Selection, sourceCode string) Selection {
	lines := strings.Split(sourceCode, "\n")
	selection.Begin = makeValid(selection.Begin, lines)
	selection.End = makeValid(selection.End, lines)
	for isWhitespace(lines, selection.Begin) &&
		!endOfLines(selection.Begin, lines) &&
		!emptySelection(selection) {
		selection.Begin = rightOf(selection.Begin, lines)
	}
	for isWhitespace(lines, leftOf(selection.End, lines)) &&
		!beginOfLines(selection.End, lines) &&
		!emptySelection(selection) {
		selection.End = leftOf(selection.End, lines)
	}
	return selection
}

func makeValid(pos Position, lines []string) Position {
	return pos
}

func isWhitespace(lines []string, pos Position) bool {
	isWhitespace := map[byte]bool{' ': true, '\t': true}
	return endOfLine(lines, pos) || isWhitespace[lines[pos.Line-1][pos.Column-1]]
}

func endOfLines(pos Position, lines []string) bool {
	return endOfLine(lines, pos) && pos.Line == len(lines)
}

func beginOfLines(pos Position, lines []string) bool {
	return pos.Column == 1 && pos.Line == 1
}

func endOfLine(lines []string, pos Position) bool {
	return pos.Column == len(lines[pos.Line-1])+1
}

func emptySelection(selection Selection) bool {
	return selection.Begin == selection.End
}

func rightOf(pos Position, lines []string) Position {
	if pos.Column == len(lines[len(lines)-1])+1 && pos.Line == len(lines) {
		return pos
	}
	if pos.Column < len(lines[pos.Line-1])+1 {
		return Position{pos.Line, pos.Column + 1}
	} else {
		return Position{pos.Line + 1, 1}
	}
}

func leftOf(pos Position, lines []string) Position {
	if pos.Column == 1 && pos.Line == 1 {
		return pos
	}
	if pos.Column > 1 {
		return Position{pos.Line, pos.Column - 1}
	} else {
		return Position{pos.Line - 1, len(lines[pos.Line-2]) + 1}
	}
}

// TODO: error handling. Do this with regex
func selectionFromString(s string) Selection {
	s = strings.Replace(s, " ", "", -1)
	beginEnd := strings.Split(s, "-")
	beginString := beginEnd[0]
	endString := beginEnd[1]
	begin := strings.Split(beginString, ":")
	end := strings.Split(endString, ":")

	return Selection{
		Begin: Position{util.ToInt(begin[0]), util.ToInt(begin[1])},
		End:   Position{util.ToInt(end[0]), util.ToInt(end[1])},
	}
}
