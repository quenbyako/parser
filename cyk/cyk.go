package cyk

import (
	"fmt"
	"os"
	"reflect"
	"runtime/debug"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/quenbyako/parser/grammar"
)

type Node interface {
	fmt.Stringer
	Name() ebnf.IdentNoEpsilon
}

func selectNode(s ebnf.Ident, n Node) bool {
	name := n.Name()

	switch s := s.(type) {
	case ebnf.UniqueIdent:
		name, ok := name.(ebnf.UniqueIdent)
		if !ok {
			return false
		}

		return s == name

	case ebnf.BaseIdent:
		name, ok := name.(ebnf.BaseIdent)
		if !ok {
			return false
		}

		if s.ID != name.ID {
			return false
		}

		// только из селектора, поскольку они самые важные
		for item := range s.Properties {
			if _, ok := name.Properties[item]; !ok {
				return false
			}
		}
		return true

	default:
		return false
	}
}

func CutGeneratedNodes(n Node) Node {
	var nodes []Node

	switch n := n.(type) {
	case Terminal:
		return n

	case SingleNonTerminal, RawNonTerminal:
		nodes = GetInnderNodes(n)

	default:
		debug.PrintStack()
		panic("unprocessable type: " + reflect.TypeOf(n).String())
	}

	for i, node := range nodes {
		nodes[i] = CutGeneratedNodes(node)
	}

	return NonTerminal{
		IdentNoEpsilon: n.Name(),
		Nodes:          nodes,
	}
}

func GetInnderNodes(n Node) []Node {
	if _, ok := n.(Terminal); ok {
		panic("can't get inner nodes of terminal")
	}

	return getInnderNodes(n)
}

func getInnderNodes(n Node) []Node {
	switch n := n.(type) {
	case Terminal:
		return []Node{n}

	case SingleNonTerminal:
		if _, ok := n.Node.Name().(ebnf.UniqueIdent); ok {
			return getInnderNodes(n.Node)
		}
		return []Node{n.Node}

	case RawNonTerminal:
		var left []Node
		if _, ok := n.Left.Name().(ebnf.UniqueIdent); ok {
			left = getInnderNodes(n.Left)
		} else {
			left = []Node{n.Left}
		}
		var right []Node
		if _, ok := n.Right.Name().(ebnf.UniqueIdent); ok {
			right = getInnderNodes(n.Right)
		} else {
			right = []Node{n.Right}
		}

		return append(left, right...)

	default:
		panic("unprocessable type: " + reflect.TypeOf(n).String())
	}
}

type SpecialRule struct {
	// селекторы более высокого правила
	selectorLeft  ebnf.Ident
	selectorRight ebnf.Ident

	// терминал, который в итоге должен получится
	mergingIdent ebnf.IdentNoEpsilon
	chainLeft    []ebnf.IdentNoEpsilon
	chainRight   []ebnf.IdentNoEpsilon
}

func (r SpecialRule) generateChain(left, right Node) Node {
	for _, ident := range r.chainLeft {
		left = SingleNonTerminal{
			IdentNoEpsilon: ident,
			Node:           left,
		}
	}
	for _, ident := range r.chainRight {
		right = SingleNonTerminal{
			IdentNoEpsilon: ident,
			Node:           right,
		}
	}

	return RawNonTerminal{
		IdentNoEpsilon: r.mergingIdent,
		Left:           left,
		Right:          right,
	}
}

type ruleDouble struct {
	ident ebnf.IdentNoEpsilon
	left  ebnf.Ident
	right ebnf.Ident
}

type ruleSingle struct {
	ident  ebnf.IdentNoEpsilon
	center ebnf.Ident
}

type Matrix struct {
	doubleRules []ruleDouble
	singleRules []ruleSingle

	table         map[xy][]Node
	maxTableIndex uint
}

func NewMatrix(terms [][]Terminal, grammar ebnf.RuleSet) *Matrix {
	sr := make([]ruleSingle, 0, len(grammar)/2)
	dr := make([]ruleDouble, 0, len(grammar)/2)
	for _, rule := range grammar {
		switch len(rule.Selectors) {
		case 1:
			sr = append(sr, ruleSingle{ident: rule.Name, center: rule.Selectors[0]})
		case 2:
			dr = append(dr, ruleDouble{ident: rule.Name, left: rule.Selectors[0], right: rule.Selectors[1]})
		default:
			panic("non CNF grammar")
		}
	}

	m := &Matrix{
		doubleRules:   dr,
		singleRules:   sr,
		table:         make(map[xy][]Node),
		maxTableIndex: uint(len(terms) - 1),
	}

	m.fillTerminals(terms)
	return m
}

func (m *Matrix) Parse() []Node {
	for i := 1; i <= int(m.maxTableIndex); i++ {
		m.CalculateLine(i)
	}

	items := m.GetEndNodes()
	for i, item := range items {
		items[i] = CutGeneratedNodes(item)
	}

	return items
}

// TODO: нужно изменить немного процесс для того, что бы мы могли пушить новые терминалы, а не вычислять линию сразу же
func (m *Matrix) CalculateLine(line int) {
	if line > int(m.maxTableIndex) {
		return
	}

	// todo: разобраться, а на кой черт нужны скобки? иначе не собирается
	for pos := (xy{uint(line), 0}); pos.x <= m.maxTableIndex; pos = (xy{pos.x + 1, pos.y + 1}) {
		m.table[pos] = m.GetPossibleSelections(pos)
	}
}

func (m *Matrix) GetPossibleSelections(pos xy) []Node {
	var res []Node

	// left является самым левым возможным в таблице значением. поскольку у нас таблица  отсечена по диагонали (с верхнего левого по нижнего правого угла)
	left, bottom := xy{pos.y, pos.y}, xy{pos.x, pos.y + 1}
	nextCoords := func() {
		left.x++
		bottom.y++
	}

	// почему bottom.x >= bottom.y: в таблице исключены все ячейки, в которых x < y, поэтому если x >= y, то мы гарантированно в самом низу
	for ; bottom.x >= bottom.y; nextCoords() {
		// итерируемся по всем элементам в исследуемых ячейках
		for _, leftOne := range m.table[left] {
			for _, bottomOne := range m.table[bottom] {
				for _, rule := range m.doubleRules {
					if !(selectNode(rule.left, leftOne) && selectNode(rule.right, bottomOne)) {
						continue
					}

					res = append(res, m.getAllSingleChainNodes(RawNonTerminal{
						IdentNoEpsilon: rule.ident,
						Left:           leftOne,
						Right:          bottomOne,
					})...)
				}
			}
		}
	}

	return res
}

func (m *Matrix) moreNodes(n Node) []Node {
	var res []Node

	for _, rule := range m.singleRules {
		if selectNode(rule.center, n) {
			res = append(res, SingleNonTerminal{
				IdentNoEpsilon: rule.ident,
				Node:           n,
			})
		}
	}
	return res
}

func (m *Matrix) getAllSingleChainNodes(n Node) []Node {
	var res []Node

	nonResearched := []Node{n}
	for len(nonResearched) > 0 {
		var newSlice []Node

		for _, node := range nonResearched {
			newSlice = append(newSlice, m.moreNodes(node)...)
		}
		res = append(res, nonResearched...)
		nonResearched = newSlice
	}

	return res
}

// GetEndNodes возвращает возможные деревья которые удалось распарсить
func (m *Matrix) GetEndNodes() []Node {
	return m.table[xy{m.maxTableIndex, 0}]
}

// PrintTable выводит дебаг информацию о том, какие ноды в ячейках находятся на текущий момент
func (m *Matrix) PrintTable() {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetRowLine(true)

	res := make([]string, m.maxTableIndex+1)
	for i := 0; i <= int(m.maxTableIndex); i++ {
		nodes := m.get(i, i)
		for _, node := range nodes {
			n, ok := node.(Terminal)
			if !ok {
				continue
			}
			res[i] = n.String()
			continue
		}
	}
	table.SetHeader(res)

	for i := 0; i <= int(m.maxTableIndex); i++ {
		table.Append(m.rowStrings(i))
	}

	table.Render()
}

func (m *Matrix) rowStrings(y int) []string {
	res := make([]string, m.maxTableIndex+1)
	// начиная с y, потому что раньше нод нет гарантированно
	for x := y; x <= int(m.maxTableIndex); x++ {
		nodes := m.get(x, y)
		more := make([]string, len(nodes))
		for j, node := range nodes {
			more[j] = node.Name().String()
		}
		res[x] = strings.Join(more, ", ")
	}

	return res
}

func (m *Matrix) fillTerminals(t [][]Terminal) {
	for i, terms := range t {
		nodes := make([]Node, 0, len(terms))
		for _, term := range terms {
			nodes = append(nodes, m.getAllSingleChainNodes(term)...)
		}

		m.push(i, i, nodes...)
	}
}

func (m *Matrix) push(x, y int, nodes ...Node) {
	m.table[xy{uint(x), uint(y)}] = append(m.get(x, y), nodes...)
}

func (m *Matrix) get(x, y int) []Node {
	nodes, ok := m.table[xy{uint(x), uint(y)}]
	if !ok {
		return []Node{}
	}
	return nodes
}
