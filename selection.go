package main

type Selection struct {
	Begin, End Position
}

type Position struct {
	Line, Column int
}
