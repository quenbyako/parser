package grammar

import (
	"fmt"
	"strings"

	"github.com/quenbyako/parser/slices"
	"golang.org/x/exp/maps"
)

const stringPadding = 4
const epsilonSymbol = "ε"

func pad(minus int) string { return strings.Repeat(" ", stringPadding-minus) }

type EBNF struct {
	Rules map[Ident][]Expr

	// те самые идентификаторы<вместе=с внутренними="аттрибутами">
	Terminals map[Ident]ComplexIdent
	// сюда помещаются все константы, которые есть в грамматике (не регулярки)
	Constants map[uint64]string
}

func (e *EBNF) String() string {
	keys := maps.Keys(e.Rules)

	slices.SortFunc(keys, func(a, b Ident) bool { return a.Cmp(b) < 0 })

	strs := make([]string, 0, len(keys))
	for _, k := range keys {
		for _, exp := range e.Rules[k] {
			strs = append(strs, fmt.Sprintf("%v ::= %v ;", k.String(), exp.String()))
		}
	}

	return strings.Join(strs, "\n")
}

func (e EBNF) AsBNF() (res *BNF) {
	res = &BNF{
		Rules:     make(RuleSet, len(e.Rules)),
		Counter:   make(IdentCounter),
		Terminals: e.Terminals,
	}
	for name, exprs := range e.Rules {
		for _, expr := range exprs {
			unwrapped, moreRules := expr.UnwrapBNF(func() Ident { return res.Counter.NewIdent(name.ID) })
			res.Rules = res.Rules.AppendRules(name, slices.Filter(unwrapped, func(s IdentSet) bool { return len(s) > 0 })...)
			res.Rules = mapsMerge(res.Rules, moreRules)
		}
	}

	return res
}

func (e EBNF) AsCNF(startRule string) *CNF { return e.AsBNF().AsCNF(startRule) }

// func (e EBNF) MergeManyProductions() EBNF {
// 	newGrammar := make(EBNF, 0, len(e))
//
// 	for _, rule := range e {
// 		newGrammar = pushNewProduction(newGrammar, rule)
// 	}
//
// 	return newGrammar
// }

// func pushNewProduction(p EBNF, item Rule) EBNF {
// 	for _, existedRule := range p {
// 		if existedRule.Name.Eq(item.Name) {
// 			switch e := existedRule.Expr.(type) {
// 			case Alts:
// 				(*p)[i].Expr = append(e, item.Expr)
// 			default:
// 				(*p)[i].Expr = Alts{e, item.Expr}
// 			}
// 			return p
// 		}
// 	}
//
// 	return append(p, item)
// }

type RuleSet map[Ident]HashSet[IdentSet]

func (r RuleSet) String() string {
	strs := make([]string, 0, len(r))

	names := slices.SortFunc(maps.Keys(r), func(a, b Ident) bool { return a.Cmp(b) < 0 })

	for _, name := range names {
		rules := r[name]

		if len(rules) == 0 {
			continue
		}

		str := name.String()
		if len(str) <= 4 {
			str += pad(len(str)) + ":"
		} else {
			str += "\n" + pad(0) + ":"
		}

		str += stringIdentSet(rules) + "\n" + pad(0) + ";"

		strs = append(strs, str)
	}

	return strings.Join(strs, "\n")
}

func stringIdentSet(s HashSet[IdentSet]) string {
	rules := slices.SortFunc(maps.Values(s), func(a, b IdentSet) bool { return a.Cmp(b) < 0 })

	str := " " + rules[0].String()

	for _, rule := range rules[1:] {
		str += "\n" + pad(0) + "| " + rule.String()
	}

	return str
}

func (r RuleSet) AppendRules(name Ident, rule ...IdentSet) RuleSet {
	if r == nil {
		r = make(RuleSet, 1)
	}

	r[name] = r[name].Append(rule...)

	return r
}

type CanonicalRule struct {
	Name Ident
	Rule IdentSet
}

func (r RuleSet) IterRules() chan CanonicalRule {
	m := make(chan CanonicalRule)
	go func(r RuleSet) {
		for name, rules := range r {
			for _, rule := range rules {
				m <- CanonicalRule{Name: name, Rule: rule}
			}
		}
		close(m)
	}(r)

	return m
}

func (r RuleSet) GetRules(name Ident) []IdentSet {
	rules, ok := r[name]
	if !ok {
		return []IdentSet{}
	}
	res := make([]IdentSet, 0, len(rules))
	for _, rule := range rules {
		res = append(res, rule)
	}

	return res
}

// ReplaceEverywhere заменяет определенный нетерминал на несколько
// последовательностей нетерминалов
func (r RuleSet) ReplaceEverywhere(id Ident, to []IdentSet) RuleSet {
	res := RuleSet{}

	for name, rules := range r {
		for _, rule := range rules {
			type variation = []IdentSet
			type ruleVariations = []variation

			more := make(ruleVariations, len(rule))
			for i, selector := range rule {
				if selector == id {
					more[i] = to
				} else {
					more[i] = []IdentSet{{selector}}
				}
			}

			for _, variantRaw := range slices.Possibles(more) {
				variant := slices.AppendMany(variantRaw...)
				if len(variant) > 0 {
					res = res.AppendRules(name, variant)
				}
			}
		}

	}

	return res
}

// BNF абсолютно отличается от грамматики:
// правила в BNF не могут состоять из альтернатив, или каких-то других типов из ebnf
// каждое правило это только сочетание терминалов или нетерминалов
type BNF struct {
	Rules     RuleSet
	Terminals map[Ident]ComplexIdent

	Counter IdentCounter
}

func (g *BNF) String() string { return g.Rules.String() }

func (g *BNF) AsCNF(startRule string) *CNF {
	allowedEmpty, found := g.ContainsEmptyRules(Ident{ID: startRule})
	if !found {
		panic("start rule not found!")
	}

	g.ExplodeLongRules()
	//fmt.Println(g)
	g.RemoveEpsilonRules()
	//fmt.Println("=====================")
	//fmt.Println(g)
	chains := g.PopChains()
	//fmt.Println("=====================")
	//fmt.Println(g)

	dualRules := make(map[Ident]HashSet[DualRule])
	stopRules := make(map[Ident]Set[Ident])
	for rule := range g.Rules.IterRules() {
		switch len(rule.Rule) {
		case 0:
			panic("found empty rule! " + fmt.Sprintf("%v : %v ;", rule.Name, epsilonSymbol))
		case 1:
			stopRules[rule.Rule[0]] = stopRules[rule.Rule[0]].Append(rule.Name)
		case 2:
			dualRules[rule.Name] = dualRules[rule.Name].Append(DualRule{rule.Rule[0], rule.Rule[1]})
		default:
			panic("got too long rule! " + fmt.Sprintf("%v : %v ;", rule.Name, rule.Rule.String()))
		}

	}

	return &CNF{
		StartRule:  startRule,
		CanBeEmpty: allowedEmpty,
		Chains:     chains,
		Rules:      dualRules,
		StopRules:  stopRules,
	}
}

func (g *BNF) ContainsEmptyRules(i Ident) (res, found bool) {
	rules, found := g.Rules[i]
	_, res = rules[emptyHash]

	return res, found
}

// убирает определенный селектор из, собственно, селекторов в правилах
func (g *BNF) AnnihilateSelector(target Ident) {
	g.Rules = g.Rules.ReplaceEverywhere(target, []IdentSet{{}})
}

type CNF struct {
	StartRule  string
	CanBeEmpty bool
	Chains     ChainList

	Rules map[Ident]HashSet[DualRule]

	// стоп правила это те, которые состоят только из одного идентификатора
	// Смысл в том, что убирая цепочные правила и сохраняя цепочки мы УЖЕ
	// заменили конечные еденичные правила в грамматиках, поэтому мы никогда их
	// не увидим
	//
	// стоп правила можно использовать при финализации результата парсера,
	// выдавая пользователю нужные нетерминалы
	//
	// ключ — результирующий терминал или нетерминал, значения — все вариации во
	// что может этот терминал разрастись
	StopRules map[Ident]Set[Ident]
}

func (g *CNF) String() string {
	rulesStr := make([]string, 0, len(g.Rules))
	keys := slices.SortEq(maps.Keys(g.Rules))

	for _, name := range keys {
		for _, rule := range slices.SortEq(maps.Values(g.Rules[name])) {
			rulesStr = append(rulesStr, fmt.Sprintf("%v -> %v %v .", name, rule[0], rule[1]))
		}
	}

	stopKeys := slices.SortEq(maps.Keys(g.StopRules))
	for _, name := range stopKeys {
		for _, ident := range slices.SortEq(maps.Keys(g.StopRules[name])) {
			rulesStr = append(rulesStr, fmt.Sprintf(". %v <- %v", name, ident))
		}
	}

	return strings.Join(rulesStr, "\n") + g.Chains.String()
}
