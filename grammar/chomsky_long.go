package grammar

// ExplodeLongRules разбивает все длинные правила на несколько коротких. Длинными считаются все правила
// содержащие более 2 селекторов.
//
// https://t.ly/-ilI
func (g *BNF) ExplodeLongRules() {
	res := make(RuleSet, len(g.Rules))

	for rule := range g.Rules.IterRules() {
		replaced, more := explodeLongRule(rule.Rule, func() Ident { return g.Counter.NewIdent(rule.Name.ID) })
		res = mapsMerge(res, more)
		res = res.AppendRules(rule.Name, replaced)
	}

	g.Rules = res
}

func explodeLongRule(r IdentSet, identGenerator func() Ident) (replaced IdentSet, moreRules RuleSet) {
	if len(r) <= 2 {
		return r, nil
	}
	explodedIdent := identGenerator()
	explodedRule, evenMore := explodeLongRule(r[1:], identGenerator)
	evenMore = evenMore.AppendRules(explodedIdent, explodedRule)

	return IdentSet{r[0], explodedIdent}, evenMore
}
