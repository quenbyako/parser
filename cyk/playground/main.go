package main

import (
	"fmt"

	"github.com/quenbyako/parser/cyk"
	"github.com/quenbyako/parser/grammar"
)

// https://www.geeksforgeeks.org/cyk-algorithm-for-context-free-grammar/
func main() {
	t := &cyk.Table{
		Data: make(map[cyk.XY][]cyk.NonTerminal),
	}

	b, bIdents := cyk.Terminal{Type: grammar.Ident{ID: "b"}}, []grammar.Ident{{ID: "B"}}
	a, aIdents := cyk.Terminal{Type: grammar.Ident{ID: "a"}}, []grammar.Ident{{ID: "A"}, {ID: "C"}}

	t.AddTerminals(b, bIdents, selector)
	fmt.Println(t.String())
	t.AddTerminals(a, aIdents, selector)
	fmt.Println(t.String())
	t.AddTerminals(a, aIdents, selector)
	fmt.Println(t.String())
	t.AddTerminals(b, bIdents, selector)
	fmt.Println(t.String())
		t.AddTerminals(a, aIdents, selector)
	fmt.Println(t.String())
}

func selector(left, bottom grammar.Ident) ([]grammar.Ident, bool) {
	switch left.ID {
	case "A":
		switch bottom.ID {
		case "B":
			return []grammar.Ident{{ID: "S"}, {ID: "C"}}, true
			// case "C":
			// 	return []grammar.Ident{{ID: "G"}}, true
		}
	case "B":
		switch bottom.ID {
		case "A":
			return []grammar.Ident{{ID: "A"}}, true
		case "C":
			return []grammar.Ident{{ID: "S"}}, true
		}
	case "C":
		switch bottom.ID {
		case "C":
			return []grammar.Ident{{ID: "B"}}, true

		}
	}

	return nil, false
}

// S -> AB | BC
// G -> AC
// A -> BA
// B -> CC
// C -> AB
//
// a -[converts]-> A + С
// b -[converts]-> B
