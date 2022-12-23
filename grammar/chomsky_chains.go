package grammar

import (
	"encoding/binary"
	"fmt"

	"github.com/quenbyako/parser/slices"
	"github.com/zeebo/xxh3"
	"golang.org/x/exp/maps"
)

// walked означает цепочку пройденых правил( если она циклична, то мы вообще ничего не добавляем)
func GetUnchained(set RuleSet, i Ident, walked []Ident) []IdentSet {
	if slices.Contains(walked, i) {
		// 	panic("chain detected! " + stringify(append(walked, i), " -> "))
		return nil
	}

	rules, ok := set[i]
	if !ok {
		panic(fmt.Sprintf("rules for %v are not found for some reason!", i))
	}

	res := make([]IdentSet, 0, len(rules))
	for _, rule := range rules {
		if isChainGenerated(rule) {
			res = append(res, GetUnchained(set, rule[0], append(walked, i))...)
			continue
		}

		res = append(res, rule)
	}

	return res
}

func isChainGenerated(rule IdentSet) bool {
	return len(rule) == 1 && (rule[0].Generated)
}

/*
// ❌❌❌❌❌❌❌❌❌❌❌❌❌❌❌❌❌❌❌❌❌❌❌❌❌❌
//
// короче что тут нужно сделать:
//
// ❌❌❌❌❌❌❌❌❌❌❌❌❌❌❌❌❌❌❌❌❌❌❌❌❌❌
//
// 1) срезка цепочных правил циклична, и уходит вглубь грамматик, нам нужно
//    докопаться до всех возможных значений, которыми мы можем заменить цепочку,
//    булево значение определяет, поменялись ли вообще хоть какие-то правила
// 2) в последовательность (ниже) мы запихиваем возможные замены КОНКРЕТНОЙ
//    новой грамматики, которую мы создали (например A->B_1->C->GH превращается
//    в A_1 = GH, + добавляем возможную последовательность для замены
//    A_1 = A->C, то есть в ытоге мы сможем восстановить последовательность
//    описанных пользователем грамматик как A->C->GH)
//
// ❌❌❌❌❌❌❌❌❌❌❌❌❌❌❌❌❌❌❌❌❌❌❌❌❌❌

type ChainSequence struct {
	GeneratedID Ident
	Chain       []Ident
}

func (g CNF) CutChainRule(rule DualRule, isTerminal func(BaseIdent) bool, identChain []Ident) (CNF, []ChainSequence, bool) {
	if !rule.IsChain(isTerminal) {
		return CNF{rule}, nil, false
	}

	res := CNF{}
	for _, rule := range g.GetRulesByIdent(rule.Name) {
		if !rule.IsChain(isTerminal) {
			res = append(res, rule)
			continue
		}

		nextIdent := rule.Selectors[0]

		if slices.ContainsEq(identChain, nextIdent) {
			panic("cyclic rule: " + stringify(append(identChain, rule.Name), " -> "))
		}

		unwrapped := g.unwrapChainRule(isTerminal, rule.Selectors[0], append(identChain, rule.Name))
		res = append(res, unwrapped...)
	}
	return res

	return g.unwrapChainRule(isTerminal, rule.Name, nil)
}


// identChain нужна что бы задетектить случайные циклические цепочки, которые
// могут оказаться в грамматике по человеческой ошибке (но если это случилось,
// тогда скорее всег это тупо баг)
func (g CNF) unwrapChainRule(isTerminal func(BaseIdent) bool, ident BaseIdent, identChain []BaseIdent) (res CNF) {
	for _, rule := range g.GetRulesByIdent(ident) {
		//if !rule.IsChain(isTerminal) {
		//	res = append(res, rule)
		//	continue
		//}

		if slices.ContainsEq(identChain, ident) {
			panic("cyclic rule: " + stringify(append(identChain, rule.Name), " -> "))
		}

		unwrapped := g.unwrapChainRule(isTerminal, rule.Selectors[0], append(identChain, rule.Name))
		res = append(res, unwrapped...)
	}
	return res
}
*/

type Chain []Ident

func (c Chain) String() string {
	if len(c) == 0 {
		return epsilonSymbol
	}

	return stringify(c, " -> ")
}

func (c Chain) Hash() (uint64, error) {
	if len(c) == 0 {
		return emptyHash, nil
	}

	res := make([]byte, 0, len(c)*8)
	for _, ident := range c {
		h, _ := ident.Hash()
		res = binary.LittleEndian.AppendUint64(res, h)
	}

	return xxh3.Hash(res), nil
}

type ChainObj struct {
	From Ident
	Chain
}

func (l ChainObj) String() string {
	return l.From.String() + " :: " + stringify(l.Chain, " -> ")
}

type ChainList HashSet[ChainObj]

func (l ChainList) GetOrGenerate(chain Chain, counter func() Ident) (Ident, ChainList) {
	h, _ := chain.Hash()
	if v, ok := l[h]; ok {
		return v.From, l
	}

	newChain := ChainObj{
		From:  counter(),
		Chain: chain,
	}

	l[h] = newChain

	return newChain.From, l
}

func (l ChainList) String() string {
	sorted := slices.SortFunc(maps.Values(l), func(a, b ChainObj) bool { return a.From.Cmp(b.From) < 0 })
	return stringify(sorted, "\n")
}

func (l ChainList) GenerateReplaces() map[Ident][]IdentSet {
	res := make(map[Ident][]IdentSet)
	for _, obj := range l {
		res[obj.Chain[0]] = append(res[obj.Chain[len(obj.Chain)-1]], IdentSet{obj.From})
	}

	return res
}

// PopChains заменяет все цепочные правила, при этом сохраняя информацию о том,
// какие цепочки конкретно были заменены.
//
// метод НЕ фильтрует сгенерированные правила, так как это можно сделать
// впоследствии если необходимо.
func (g *BNF) PopChains() ChainList {
	res := make(RuleSet)
	chains := make(ChainList)

	for name, rules := range g.Rules {
		for _, rule := range rules {
			if !rule.isChain(slices.ToMap(maps.Keys(g.Terminals))) {
				res = res.AppendRules(name, rule)
				continue
			}
			if name == rule[0] {
				// 100% скип потому что на кой черт нам вообще правила вида
				// `S : S;`, это бред
				continue
			}

			res, chains = g.getAllChainVariations(res, chains, []Ident{name, rule[0]})
		}
	}

	for from, to := range chains.GenerateReplaces() {
		res = res.ReplaceEverywhere(from, to)
	}

	g.Rules = res

	return chains
}

func (g *BNF) getAllChainVariations(res RuleSet, chains ChainList, chain Chain) (RuleSet, ChainList) {
	lastItem := chain[len(chain)-1]
	rules, ok := g.Rules[lastItem]
	if !ok {
		panic(fmt.Sprintf("ident %q not found", lastItem))
	}

	for _, rule := range rules {
		if rule.isChain(slices.ToMap(maps.Keys(g.Terminals))) {
			if !slices.ContainsEq(chain, rule[0]) {
				res, chains = g.getAllChainVariations(res, chains, append(chain, rule[0]))
			}
			continue
		}

		var newIdent Ident
		newIdent, chains = chains.GetOrGenerate(chain, func() Ident { return g.Counter.NewIdent(chain[0].ID) })
		res = res.AppendRules(newIdent, rule)
	}

	return res, chains
}
