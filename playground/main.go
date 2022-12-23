package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/quenbyako/parser/grammar"
	"github.com/quenbyako/parser/slices"
)

func main() {
	data, err := os.Open("/Users/quenbyako/Documents/exported/code/parser/playground/regex.ebnf")
	if err != nil {
		panic(err)
	}
	defer data.Close()

	g, err := grammar.Parse("some_file", data, "string", "space")
	if err != nil {
		panic(err)
	}

	fmt.Println(g.AsCNF("regex"))

}

var epsilonTest = grammar.BNF{
	Rules: generateRuleSet([]grammar.CanonicalRule{
		{Name: id("S"), Rule: grammar.IdentSet{id("A"), id("B"), id("C")}},
		{Name: id("S"), Rule: grammar.IdentSet{id("D"), id("S")}},
		{Name: id("A"), Rule: grammar.IdentSet{}},
		{Name: id("B"), Rule: grammar.IdentSet{id("A"), id("C")}},
		{Name: id("C"), Rule: grammar.IdentSet{}},
		{Name: id("D"), Rule: grammar.IdentSet{id("d")}},
	}),
	Terminals: nil,
}

var popChainsTest = parseExample([]string{"d", "b", "x", "y"}, `
	S    : gen1
	     ;
	gen1 : gen2 C b
	     | gen1 A
	     | gen1
	     | A
	     | gen2
	     ;
	gen2 : d b
	     ;
	A    : x y
	     | x
	     ;
    C    : d x
	     ;
`)

func id(i string) grammar.Ident {
	return grammar.Ident{ID: i, Generated: false}
}

func stringify[S ~[]T, T fmt.Stringer](s S, sep string) string {
	return strings.Join(slices.Remap(s, func(_ int, v T) string { return v.String() }), sep)
}

func generateRuleSet(rules []grammar.CanonicalRule) grammar.RuleSet {
	set := make(grammar.RuleSet)
	for _, rule := range rules {
		set = set.AppendRules(rule.Name, rule.Rule)
	}

	return set
}

func parseExample(terms []string, text string) *grammar.BNF {
	g, err := grammar.Parse("", strings.NewReader(text), terms...)
	if err != nil {
		panic(err)
	}

	return g.AsBNF()
}
