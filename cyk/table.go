package cyk

import (
	"bytes"
	"fmt"
	"math"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/quenbyako/parser/grammar"
	"github.com/quenbyako/parser/slices"
	"github.com/takuoki/clmconv"
)

// XY is a alias to 2 dimension coordinates for maps inside parser matrix
type XY struct{ X, Y int }

func termxy(i int) XY { return XY{X: i, Y: i} }

func (n XY) String() string { return fmt.Sprintf("coords[%v:%v]", clmconv.Itoa(n.X), n.Y) }

// координаты:
// O X →
// Y
// ↓   ↘

type Table struct {
	terms []Terminal
	Data  map[XY][]NonTerminal
}

func (t *Table) String() string {
	buf := bytes.NewBuffer(nil)
	w := tablewriter.NewWriter(buf)

	terms := slices.Remap(t.terms, func(_ int, t Terminal) string { return t.Type.String() })
	w.SetHeader(terms)

	for y := range t.terms {
		row := make([]string, len(t.terms))
		for x := range t.terms {
			row[x] = strings.Join(slices.Remap(t.Data[XY{X: x, Y: y}], func(_ int, n NonTerminal) string { return n.I.String() }), ",")
		}
		w.Append(row)
	}

	w.Render()

	return buf.String()
}

func (t *Table) AddTerminals(term Terminal, nonterms []grammar.Ident, selector selectorFunc) {
	t.terms = append(t.terms, term)
	t.Data[termxy(len(t.terms)-1)] = slices.Remap(nonterms, func(_ int, i grammar.Ident) NonTerminal {
		return NonTerminal{
			I: i,
			// просто что бы если была попытка через него найти нетерминал то
			// улететь в панику
			Left:   NonTerminalCoord{XY: termxy(math.MaxInt), Index: -1000},
			Bottom: NonTerminalCoord{XY: termxy(math.MaxInt), Index: -1000},
		}
	})

	t.recalculateLine(len(t.terms)-1, selector)

}

func (t *Table) recalculateLine(i int, selector selectorFunc) {
	currentCell := XY{X: i, Y: i - 1}
	for ; currentCell.Y >= 0; currentCell.Y-- {
		t.FillCell(currentCell, selector)
	}
}

type selectorFunc = func(left, bottom grammar.Ident) ([]grammar.Ident, bool)

func (t *Table) FillCell(cell XY, selector selectorFunc) {
	if cell.Y > cell.X {
		panic("out of bounds")
	}
	if cell.Y == cell.X {
		panic(fmt.Sprintf("%v must be filled right after adding nonterm", cell))
	}

	// самая левая ячейка будет y потому что на каждую колонку мы смещаемся на 1
	LeftCell := XY{X: cell.Y, Y: cell.Y}
	// первая нижняя ячейка будет по иксу та же, по игреку +1
	BottomCell := XY{X: cell.X, Y: cell.Y + 1}
	// проходимся по каждой ячейке от самой левой до coord.x-1 (потому что себя нет смысла проверять)
	var next = func() {
		LeftCell.X++
		BottomCell.Y++
	}

	resultedTerms := make([]NonTerminal, 0)
	for ; LeftCell.X < cell.X && BottomCell.Y <= cell.X; next() {
		for leftIndex, leftNode := range t.Data[LeftCell] {
			for bottomIndex, bottomNode := range t.Data[BottomCell] {
				if newIdents, ok := selector(leftNode.I, bottomNode.I); ok {
					resultedTerms = append(resultedTerms,
						slices.Remap(newIdents, func(_ int, i grammar.Ident) NonTerminal {
							return NonTerminal{
								I:      i,
								Left:   NonTerminalCoord{XY: LeftCell, Index: leftIndex},
								Bottom: NonTerminalCoord{XY: BottomCell, Index: bottomIndex},
							}
						})...,
					)
				}
			}
		}
	}

	t.Data[cell] = resultedTerms
}
