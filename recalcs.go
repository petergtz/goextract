package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"reflect"
	"strings"

	"github.com/petergtz/goextract/util"
)

func RecalcPoses(node ast.Node, pos token.Pos, offset *token.Pos, indent int) {
	if node == nil {
		return
	}
	if node.Pos() != token.NoPos && offset != nil {
		// if *offset != token.NoPos && *offset != pos-node.Pos() {
		// 	panic("Nested sub-node with already set poses must all have same offset. Existing offset:" + spew.Sdump(offset) + " vs new offset: " + spew.Sdump(pos-node.Pos()))
		// }
		if *offset == token.NoPos {
			*offset = pos - node.Pos()
		}
		shiftPoses(node, *offset)
		return
	}
	switch typedNode := node.(type) {
	case *ast.File:
		typedNode.Package = pos
		// TODO do import statements
		RecalcPoses(typedNode.Name, pos+8, offset, indent)
		currentPos := typedNode.Name.End() + 2
		for _, decl := range typedNode.Decls {
			RecalcPoses(decl, currentPos, offset, indent)
			currentPos = decl.End() + 1
		}
	case *ast.FuncDecl:
		typedNode.Type.Func = pos
		if typedNode.Recv != nil {
			RecalcPoses(typedNode.Recv, pos+5, offset, indent)
			pos = typedNode.Recv.End() + 1
		} else {
			pos += 5
		}
		RecalcPoses(typedNode.Name, pos, offset, indent)
		pos = typedNode.Name.End()
		RecalcPoses(typedNode.Type.Params, pos, offset, indent)
		pos = typedNode.Type.Params.End() + 1

		if typedNode.Type.Results != nil {
			RecalcPoses(typedNode.Type.Results, pos, offset, indent)
			pos = typedNode.Type.Results.End() + 1

		}
		RecalcPoses(typedNode.Body, pos, offset, indent)
	case *ast.FieldList:
		typedNode.Opening = pos
		pos++
		for _, field := range typedNode.List {
			RecalcPoses(field, pos, offset, indent)
			pos = field.End() + 1

		}
		if len(typedNode.List) > 0 {
			pos--
		}
		typedNode.Closing = pos

	case *ast.Field:
		if typedNode.Tag != nil {
			panic("Field tags not supported yet")
		}
		for _, name := range typedNode.Names {
			RecalcPoses(name, pos, offset, indent)
			pos = name.End() + 1
		}
		RecalcPoses(typedNode.Type, pos, offset, indent)

	case *ast.BlockStmt:
		typedNode.Lbrace = pos
		pos += 2 + token.Pos(indent) + 1
		for _, stmt := range typedNode.List {
			RecalcPoses(stmt, pos, offset, indent+1)
			pos = stmt.End() + 1 + token.Pos(indent) + 1 // new line for next stmt
		}
		typedNode.Rbrace = pos

	case *ast.ReturnStmt:
		typedNode.Return = pos
		pos += 7
		for _, result := range typedNode.Results {
			RecalcPoses(result, pos, offset, indent)
			pos = result.End() + 1
		}

	case *ast.GenDecl:
		typedNode.TokPos = pos
		typedNode.Lparen = pos + token.Pos(len(typedNode.Tok.String())) + 1
		currentPos := typedNode.Lparen + 1 + 1 /* new line*/ + 1 /*indentation*/

		for _, spec := range typedNode.Specs {
			RecalcPoses(spec, currentPos, offset, indent)
			currentPos = spec.End() + 1
		}
		typedNode.Rparen = currentPos

	case *ast.ValueSpec:
		currentPos := pos

		for _, name := range typedNode.Names {
			RecalcPoses(name, currentPos, offset, indent)
			currentPos = name.End() + 1
		}
		for _, value := range typedNode.Values {
			RecalcPoses(value, currentPos, offset, indent)
			currentPos = value.End() + 1
		}
		RecalcPoses(typedNode.Type, currentPos, offset, indent)

	case *ast.Ident:
		typedNode.NamePos = pos
	case *ast.Ellipsis:
		typedNode.Ellipsis = pos
		RecalcPoses(typedNode.Elt, pos+3, offset, indent)
	case *ast.BasicLit:
		typedNode.ValuePos = pos
	case *ast.FuncLit:
		RecalcPoses(typedNode.Type, pos, offset, indent)
		RecalcPoses(typedNode.Body, typedNode.Type.End()+1, offset, indent)
	case *ast.CompositeLit:
		RecalcPoses(typedNode.Type, pos, offset, indent)
		typedNode.Lbrace = typedNode.Type.End() + 1
		currentPos := typedNode.Type.End() + 2
		for _, elt := range typedNode.Elts {
			RecalcPoses(elt, currentPos, offset, indent)
			currentPos = elt.End() + 1
		}
		typedNode.Rbrace = currentPos
	case *ast.ParenExpr:
		typedNode.Lparen = pos
		RecalcPoses(typedNode.X, pos+1, offset, indent)
		typedNode.Rparen = typedNode.X.End() + 1
	case *ast.CallExpr:
		RecalcPoses(typedNode.Fun, pos, offset, indent)
		typedNode.Lparen = typedNode.Fun.End()
		currentPos := typedNode.Lparen + 1
		for _, arg := range typedNode.Args {
			RecalcPoses(arg, currentPos, offset, indent)
			currentPos = arg.End() + 1
		}
		// TODO: is this the right treatment in all cases?
		typedNode.Ellipsis = token.NoPos
		typedNode.Rparen = currentPos
	case *ast.BinaryExpr:
		RecalcPoses(typedNode.X, pos, offset, indent)
		typedNode.OpPos = typedNode.X.End() + 1
		RecalcPoses(typedNode.Y, typedNode.OpPos+token.Pos(len(typedNode.Op.String())), offset, indent)
	case *ast.FuncType:
	// not needed because we do above
	case *ast.StarExpr:
		typedNode.Star = pos
		RecalcPoses(typedNode.X, pos+1, offset, indent)
	case *ast.ExprStmt:
		RecalcPoses(typedNode.X, pos, offset, indent)
	case *ast.AssignStmt:

		for _, lhs := range typedNode.Lhs {
			RecalcPoses(lhs, pos, offset, indent)
			// TODO: should this increment by 2 to include space after comma?
			pos = lhs.End() + 1
		}
		typedNode.TokPos = pos
		pos += token.Pos(len(typedNode.Tok.String()))
		for _, rhs := range typedNode.Rhs {
			RecalcPoses(rhs, pos, offset, indent)
			pos = rhs.End() + 1
		}
	case *ast.SelectorExpr:
		RecalcPoses(typedNode.X, pos, offset, indent)
		RecalcPoses(typedNode.Sel, typedNode.X.End()+1, offset, indent)
	case *ast.ArrayType:
		typedNode.Lbrack = pos
		pos += 2
		// TODO treat real arrays correctly
		// RecalcPoses(typedNode.Len, pos, offset)
		RecalcPoses(typedNode.Elt, pos, offset, indent)
	default:
		panic(fmt.Sprintf("recalcing not implemented for %v yet", reflect.TypeOf(node)))

	}
}

func shiftPoses(node ast.Node, offset token.Pos) {
	ast.Inspect(node, func(node ast.Node) bool {
		if node == nil {
			return true
		}
		shiftPosesNonRecursively(node, offset, 0)
		return true
	})
}

func getPosesForNode(node ast.Node) []*token.Pos {
	switch typedNode := node.(type) {
	case *ast.Ident:
		return []*token.Pos{&typedNode.NamePos}
	case *ast.File:
		return []*token.Pos{&typedNode.Package}
	case *ast.FuncDecl:
		return []*token.Pos{}
	case *ast.FieldList:
		return []*token.Pos{&typedNode.Opening, &typedNode.Closing}
	case *ast.Field:
		return []*token.Pos{}
	case *ast.BlockStmt:
		return []*token.Pos{&typedNode.Lbrace, &typedNode.Rbrace}
	case *ast.ReturnStmt:
		return []*token.Pos{&typedNode.Return}
	case *ast.AssignStmt:
		return []*token.Pos{&typedNode.TokPos}
	case *ast.GenDecl:
		return []*token.Pos{&typedNode.TokPos, &typedNode.Lparen, &typedNode.Rparen}
	case *ast.Ellipsis:
		return []*token.Pos{&typedNode.Ellipsis}
	case *ast.BasicLit:
		return []*token.Pos{&typedNode.ValuePos}
	case *ast.CompositeLit:
		return []*token.Pos{&typedNode.Lbrace, &typedNode.Rbrace}
	case *ast.ParenExpr:
		return []*token.Pos{&typedNode.Lparen, &typedNode.Rparen}
	case *ast.ValueSpec:
		return []*token.Pos{}
	case *ast.SelectorExpr:
		return []*token.Pos{}
	case *ast.IndexExpr:
		return []*token.Pos{&typedNode.Lbrack, &typedNode.Rbrack}
	case *ast.SliceExpr:
		return []*token.Pos{&typedNode.Lbrack, &typedNode.Rbrack}
	case *ast.CallExpr:
		return []*token.Pos{&typedNode.Lparen, &typedNode.Ellipsis, &typedNode.Rparen}
	case *ast.BinaryExpr:
		return []*token.Pos{&typedNode.OpPos}
	case *ast.FuncType:
		return []*token.Pos{&typedNode.Func}
	case *ast.ImportSpec:
		return []*token.Pos{&typedNode.EndPos}
	case *ast.ExprStmt:
		return []*token.Pos{}
	case *ast.RangeStmt:
		return []*token.Pos{&typedNode.For, &typedNode.TokPos}
	case *ast.ArrayType:
		return []*token.Pos{&typedNode.Lbrack}
	case *ast.TypeSpec:
		return []*token.Pos{}
	case *ast.StructType:
		return []*token.Pos{&typedNode.Struct}
	case *ast.UnaryExpr:
		return []*token.Pos{&typedNode.OpPos}
	case *ast.StarExpr:
		return []*token.Pos{&typedNode.Star}
	case *ast.IfStmt:
		return []*token.Pos{&typedNode.If}
	case *ast.TypeSwitchStmt:
		return []*token.Pos{&typedNode.Switch}
	case *ast.TypeAssertExpr:
		return []*token.Pos{&typedNode.Lparen, &typedNode.Rparen}
	case *ast.CaseClause:
		return []*token.Pos{&typedNode.Case, &typedNode.Colon}
	case *ast.DeclStmt:
		return []*token.Pos{}
	case *ast.MapType:
		return []*token.Pos{&typedNode.Map}
	case *ast.KeyValueExpr:
		return []*token.Pos{&typedNode.Colon}
	case *ast.BranchStmt:
		return []*token.Pos{&typedNode.TokPos}
	case *ast.CommentGroup:
		return []*token.Pos{}
	case *ast.Comment:
		return []*token.Pos{&typedNode.Slash}

	default:
		panic(fmt.Sprintf("poses not implemented for %v yet", reflect.TypeOf(node)))
	}

}

func shiftPosesNonRecursively(node ast.Node, offset token.Pos, lowerBound token.Pos) {
	for _, pos := range getPosesForNode(node) {
		if *pos > lowerBound {
			*pos += offset
		}

	}
}

func resetPoses(node ast.Node) {
	ast.Inspect(node, func(node ast.Node) bool {
		if node == nil {
			return true
		}
		for _, pos := range getPosesForNode(node) {
			*pos = 0
		}
		return true
	})
}

func lineLengthsFrom(fileSet *token.FileSet) []int {
	return lineLengthsFromLines(strings.Split(util.ReadFileAsStringOrPanic(fileSet.File(1).Name()), "\n"))
}

func lineLengthsFromLines(lines []string) []int {
	lineLengths := make([]int, len(lines))
	for i, line := range lines {
		lineLengths[i] = len(line)
	}
	return lineLengths
}

func replacementModifications(fileSet *token.FileSet, oldPos, oldEnd, newEnd token.Pos, lineLengths []int, areaRemoved []Range) (lineNum, numLinesToCut, newLength int) {
	oldEndLine := fileSet.Position(oldEnd).Line
	oldPosLine := fileSet.Position(oldPos).Line
	numCutLines := oldEndLine - oldPosLine
	newLinelen := areaRemoved[0].begin + (lineLengths[oldEndLine-1] - areaRemoved[len(areaRemoved)-1].end) + int(newEnd-oldPos)
	return oldPosLine - 1, numCutLines, newLinelen
}

func ConvertLineOffsetsToLineLengths(offsets []int, endPos int) []int {
	result := make([]int, len(offsets))
	lastOffset := 0
	for i, offset := range offsets[1:] {
		result[i] = offset - lastOffset - 1
		lastOffset = offset
	}
	result[len(result)-1] = int(endPos) - lastOffset
	return result
}

func ConvertLineLengthsToLineOffsets(lineLengths []int) []int {
	result := make([]int, len(lineLengths))
	result[0] = 0
	offset := 0
	for i, lineLength := range lineLengths[:len(lineLengths)-1] {
		result[i+1] = offset + lineLength + 1
		offset += lineLength + 1
	}
	return result
}

func shiftPosesAfterPos(node ast.Node, newNode ast.Node, pos token.Pos, by token.Pos) {
	// TODO: this must also move comments
	// ^^^ It looks like this is done
	var offset token.Pos
	visitedCommentGroups := make(map[*ast.CommentGroup]bool)
	ast.Inspect(node, func(node ast.Node) bool {
		if node == nil {
			return true
		}
		if node == newNode {
			return false
		}
		if node.End() > pos && offset == 0 {
			offset = by
		}
		if commentGroup, ok := node.(*ast.CommentGroup); ok {
			visitedCommentGroups[commentGroup] = true
		}
		shiftPosesNonRecursively(node, offset, pos)
		return true
	})
	for _, commentGroup := range node.(*ast.File).Comments {
		for _, comment := range commentGroup.List {
			if !visitedCommentGroups[commentGroup] && comment.Slash > pos {
				comment.Slash += by
			}
		}
	}

}

func insertionModifications(astFile *ast.File, funcDecl *ast.FuncDecl, areaRemoved []Range) (areaToBeAppended []int) {
	linelengths := make([]int, len(areaRemoved)-1)
	for i, line := range areaRemoved[1:] {
		linelengths[i] = line.end
	}
	result := []int{
		int(funcDecl.Body.Pos() - funcDecl.Type.Func + 1),
	}
	if funcDecl.Type.Results == nil {
		result = append(result, 1+areaRemoved[0].end-areaRemoved[0].begin)
	} else {
		result = append(result, 1+len(token.RETURN.String())+1+areaRemoved[0].end-areaRemoved[0].begin)
	}
	result = append(result, linelengths...)
	result = append(result, 1)
	return result
}

func insertionModificationsForStmts(astFile *ast.File, funcDecl *ast.FuncDecl, areaRemoved []Range, varsUsedAfterwards []ast.Expr) (areaToBeAppended []int) {
	linelengths := make([]int, len(areaRemoved))
	for i, line := range areaRemoved {
		linelengths[i] = line.end
	}
	result := []int{
		int(funcDecl.Body.Pos() - funcDecl.Type.Func + 1),
	}
	result = append(result, linelengths...)
	// TODO: this if could type assert the last stmt to see if it is a return stmt
	if len(varsUsedAfterwards) != 0 {
		vars := funcDecl.Body.List[len(funcDecl.Body.List)-1].(*ast.ReturnStmt).Results
		result = append(result, 1+len(token.RETURN.String())+1+int(vars[len(vars)-1].End()-vars[0].Pos()))
	}
	result = append(result, 1)
	return result
}

// TODO this moves all comments instead of just correct ones.
func moveComments(astFile *ast.File, moveOffset token.Pos, pos, end token.Pos) {
	for _, commentGroup := range astFile.Comments {
		for _, comment := range commentGroup.List {
			if comment.Slash >= pos && comment.Slash <= end {
				comment.Slash += moveOffset
			}
		}
	}
}

type Range struct {
	begin, end int
}

func areaRemoved(fileSet *token.FileSet, pos, end token.Pos) []Range {
	lineLengths := lineLengthsFrom(fileSet)
	b := fileSet.Position(pos)
	e := fileSet.Position(end)
	result := make([]Range, e.Line-b.Line+1)
	if e.Line > b.Line {
		result[0].begin = b.Column - 1
		result[0].end = lineLengths[b.Line-1]
		for i := 1; i < len(result)-1; i++ {
			result[i].begin = 0
			result[i].end = lineLengths[b.Line-1+i]
		}
		if len(result) > 1 {
			result[len(result)-1].begin = 0
			result[len(result)-1].end = e.Column - 1
		}
	} else {
		result[0].begin = b.Column - 1
		result[0].end = e.Column - 1
	}
	return result
}

func FakeContentForLineLengths(linesLengths []int) string {
	result := ""
	for _, linesLength := range linesLengths {
		for i := 0; i < linesLength; i++ {
			result += "x"
		}
		result += "\n"
	}
	return result
}
