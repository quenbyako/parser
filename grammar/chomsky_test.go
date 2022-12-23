package grammar_test

import (
	"fmt"
	"testing"

	. "github.com/quenbyako/parser/ebnf/parser"
	"github.com/quenbyako/parser/slices"
	"github.com/stretchr/testify/require"
)

func TestRuleSet_Cut(t *testing.T) {
	for _, tt := range []struct {
		name     string
		grammar  CNF
		expected CNF
	}{{
		// A_1 = B_1  |  a
		// B_1 = C_1  |  b
		// C_1 = D D |  c
		grammar: CNF{
			single(uniq("A", 1), uniq("B", 1)),
			single(uniq("A", 1), ident("a")),
			single(uniq("B", 1), uniq("C", 1)),
			single(uniq("B", 1), ident("b")),
			double(uniq("C", 1), ident("D"), ident("D")),
			single(uniq("C", 1), ident("c")),
		},
		// A_1 = a | b | c | 44
		// B_1 = b | c | 44
		// C_1 = c | 44
		expected: CNF{
			single(uniq("A", 1), ident("a")),
			single(uniq("A", 1), ident("b")),
			single(uniq("A", 1), ident("c")),
			double(uniq("A", 1), ident("D"), ident("D")),
			single(uniq("B", 1), ident("b")),
			single(uniq("B", 1), ident("c")),
			double(uniq("B", 1), ident("D"), ident("D")),
			single(uniq("C", 1), ident("c")),
			double(uniq("C", 1), ident("D"), ident("D")),
		},
	}} {
		t.Run(tt.name, func(t *testing.T) {
			//for _, g := range tt.grammar {
			//	fmt.Println(g.String(), g.AllowedToUnchain(nil))
			//}
			//return
			//
			//fmt.Println(tt.grammar.String())
			//fmt.Println("===================")
			res := tt.grammar.CutChainRules(nil)
			slices.SortFunc(res, func(a, b DualRule) bool { return a.Name.String() < b.Name.String() })

			fmt.Println(res.String())

		})
	}
}

func TestIdent_IsChain(t *testing.T) {
	for _, tt := range []struct {
		name       string
		rule       DualRule
		isTerminal func(ComplexIdent) bool
		expected   bool
	}{{
		rule:     single(ident("A"), uniq("B", 1)),
		expected: true,
	}, {
		rule:     double(ident("A"), uniq("B", 1), uniq("B", 1)),
		expected: false,
	}, {
		rule:     single(ident("A"), ConstIdent("")),
		expected: false,
	}, {
		rule:       single(ident("A"), ident("B")),
		isTerminal: func(i ComplexIdent) bool { return i.Eq(ComplexIdent{ID: "C"}) },
		expected:   true,
	}, {
		rule:       single(ident("A"), ident("C")),
		isTerminal: func(i ComplexIdent) bool { return i.Eq(ComplexIdent{ID: "C"}) },
		expected:   false,
	}} {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, tt.rule.IsChain(tt.isTerminal))
		})
	}
}

func single(i, s Ident) DualRule      { return DualRule{Name: i, Selectors: [2]Ident{s}} }
func double(i, s1, s2 Ident) DualRule { return DualRule{Name: i, Selectors: [2]Ident{s1, s2}} }

func ident(s string) Ident         { return ComplexIdent{ID: s} }
func uniq(ref string, i int) Ident { return UniqueIdent{Ref: ref, Index: i} }
