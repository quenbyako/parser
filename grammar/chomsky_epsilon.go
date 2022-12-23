package grammar

import (
	"strconv"

	"github.com/quenbyako/parser/slices"
	"golang.org/x/exp/maps"
)

// CutEpsilon ищет все правила, в которых содержится эпсилон-правила, и преобразовывает эти правила в такие, в
// которых не будет этих эпсилон правил.
//
// https://t.ly/xM1u
func (g *BNF) RemoveEpsilonRules() {
	terms := slices.ToMap(maps.Keys(g.Terminals))

	potentiallyEmpty := g.FindEpsilon(terms)

	newSet := make(RuleSet, len(g.Rules))
	for name, rules := range g.Rules {
		if len(rules) == 0 {
			continue
		}

		for _, rule := range rules {
			variants := slices.Remap(rule, func(_ int, i Ident) IdentSet {
				if _, ok := potentiallyEmpty[i]; !ok {
					return []Ident{i}
				}
				return []Ident{{ID: epsilonSymbol}, i}
			})

			for _, replaced := range slices.Possibles(variants) {
				filtered := slices.Filter(replaced, func(i Ident) bool { return i.ID != epsilonSymbol })
				if len(filtered) > 0 {
					newSet = newSet.AppendRules(name, filtered)
				}
			}
		}
	}

	// filter completely empty rules
	g.Rules = filterCompleteEmpty(newSet, terms)
}

func (g BNF) FindEpsilon(terminals Set[Ident]) Set[Ident] {
	indexes := g.fillEpsilonIndex(terminals)

	researchQueue := indexes.popQueue()
	for len(researchQueue) > 0 {
		for item := range researchQueue {
			indexes.decreaseCounter(item)
		}
		researchQueue = indexes.popQueue()
	}

	return indexes.res
}

type counterKey struct {
	id        Ident
	ruleIndex int
}

func (c counterKey) String() string { return c.id.String() + "_" + strconv.Itoa(c.ruleIndex) }

type epsilonIndex struct {
	m   map[Ident]identIndexes
	res Set[Ident]
}

type identIndexes struct {
	isEpsilon bool
	// для каждого идентификатора будем хранить список номеров тех правил, в правой части которых он встречается
	concernedRules Set[counterKey]

	counters []int
}

// режет правила так, что бы можно было отфильтровать полностью путсые нетерминалы
//
//	S   : B
//	    | B C
//	    | A B
//	    | A B C
//	    | D S
//	    ;
//	D   : some_term ;
//
// превращается в
//
//	S   : D S ;
//	D   : some_term ;
func filterCompleteEmpty(ruleset RuleSet, terms Set[Ident]) RuleSet {
	res := make(RuleSet, len(ruleset))

	confirmedEmpty := make(Set[Ident])
	for rule := range ruleset.IterRules() {
		filtered := slices.Filter(rule.Rule, func(i Ident) bool { return !isRuleEmpty(ruleset, i, terms, confirmedEmpty) })
		if len(filtered) > 0 {
			res = res.AppendRules(rule.Name, filtered)
		}
	}

	return res
}

func isRuleEmpty(ruleset RuleSet, i Ident, terms, confirmed Set[Ident]) bool {
	if terms.Has(i) {
		return false
	}
	if confirmed.Has(i) {
		return true
	}

	rules, ok := ruleset[i]
	if !ok {
		confirmed[i] = struct{}{}
		return true
	}

	for _, rule := range rules {
		for _, selector := range rule {
			if !isRuleEmpty(ruleset, selector, terms, confirmed) {
				return false
			}
		}
	}

	return true
}

func (g *BNF) fillEpsilonIndex(terminals Set[Ident]) *epsilonIndex {
	res := &epsilonIndex{
		m:   make(map[Ident]identIndexes, len(g.Rules)),
		res: make(Set[Ident]),
	}

	for name := range g.Rules {
		rules := g.Rules.GetRules(name)

		res.setCounters(name, rules)
		res.setConcernRules(name, rules, terminals)
		// возможно что наше основное правило нигде не встречалось, но мы удалим такие ПОСЛЕ удаления эпсилон правил
		res.set(name, func(i *identIndexes) {})
	}

	return res
}

func (i *epsilonIndex) popQueue() Set[Ident] {
	queue := Set[Ident]{}
	for id, c := range i.m {
		if slices.Contains(c.counters, 0) && !c.isEpsilon {
			queue[id] = struct{}{}
			i.setAsEpsilon(id)
			i.res[id] = struct{}{}
		}
	}

	return queue
}

func (i *epsilonIndex) setAsEpsilon(id Ident) {
	i.set(id, func(i *identIndexes) { i.isEpsilon = true })
}

func (i *epsilonIndex) setCounters(id Ident, rules []IdentSet) {
	i.set(id, func(i *identIndexes) {
		i.counters = slices.Remap(rules, func(_ int, i IdentSet) int { return len(i) })
	})
}

func (i *epsilonIndex) decreaseCounter(id Ident) {
	for concerned := range i.m[id].concernedRules {
		i.m[concerned.id].counters[concerned.ruleIndex]--
	}
}

func (i *epsilonIndex) setConcernRules(id Ident, rules []IdentSet, terminals Set[Ident]) {
	// добавляем во все идентификаторы индекс правила, где этот
	// идентификатор используется
	//
	// Логика:
	//     S : ABC
	//     S : DS
	//
	// в А в эпсилон-индексе добавляем в concernedRules S и индекс 0.
	// То же самое для B и C
	// в D в эпсилон-индексе добавляем в concernedRules S и индекс 1
	// в S в эпсилон-индексе добавляем в concernedRules S и индекс 1
	for ruleIndex, rule := range rules {
		for _, selector := range rule {
			if _, ok := terminals[selector]; ok {
				continue
			}

			i.set(selector, func(i *identIndexes) {
				i.concernedRules[counterKey{id: id, ruleIndex: ruleIndex}] = struct{}{}
			})
		}
	}
}

func (i *epsilonIndex) set(id Ident, f func(i *identIndexes)) {
	x, ok := i.m[id]
	if !ok {
		x = identIndexes{
			concernedRules: Set[counterKey]{},
		}
	}
	f(&x)
	i.m[id] = x
}
