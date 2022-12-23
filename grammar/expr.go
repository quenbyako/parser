package grammar

import (
	"fmt"
	"strings"

	"github.com/quenbyako/parser/slices"
)

type Expr interface {
	fmt.Stringer
	expr()

	// http://lampwww.epfl.ch/teaching/archive/compilation-ssc/2000/part4/parsing/node3.html
	//
	// replaces это список "взорваных" правил исходящих из данного правила
	// newRules это список новых правил (работает только для повторов)
	UnwrapBNF(identGenerator func() Ident) (replaces []IdentSet, newRules RuleSet)
}

type Group struct{ E Expr }

var _ Expr = Group{}

func (_ Group) expr()                                          {}
func (g Group) String() string                                 { return "( " + g.E.String() + " )" }
func (g Group) UnwrapBNF(c func() Ident) ([]IdentSet, RuleSet) { return g.E.UnwrapBNF(c) }

type Option struct{ E Expr }

var _ Expr = Option{}

func (_ Option) expr()          {}
func (o Option) String() string { return "[ " + o.E.String() + " ]" }
func (o Option) UnwrapBNF(c func() Ident) (replaces []IdentSet, newRules RuleSet) {
	exploded, moreRules := o.E.UnwrapBNF(c)
	return append(exploded, IdentSet{}), moreRules
}

type Repeat struct{ E Expr }

var _ Expr = Repeat{}

func (_ Repeat) expr()          {}
func (r Repeat) String() string { return "{ " + r.E.String() + " }" }

// http://lampwww.epfl.ch/teaching/archive/compilation-ssc/2000/part4/parsing/node3.html
func (r Repeat) UnwrapBNF(c func() Ident) (replaces []IdentSet, newRules RuleSet) {
	// Convert every repetition { A | B | C } to a fresh non-terminal X and add
	// X = ε | X A | X B | X C.
	unwrapped, newRules := r.E.UnwrapBNF(c)
	newID := c()
	more := make([]IdentSet, len(unwrapped)+1)
	more[0] = make(IdentSet, 0) // empty rule MUST be added, it's not a replace
	for i, alternative := range unwrapped {
		more[i+1] = append(IdentSet{newID}, alternative...)
	}
	newRules = newRules.AppendRules(newID, more...)

	return []IdentSet{{newID}}, newRules
}

type Seq []Expr

var _ Expr = Seq{}

func (_ Seq) expr()          {}
func (s Seq) String() string { return stringify(s, " ") }
func (s Seq) UnwrapBNF(c func() Ident) (_ []IdentSet, newRules RuleSet) {
	newRules = make(RuleSet)

	exploded := slices.Remap(s, func(_ int, e Expr) []IdentSet {
		evenExploded, evenMoreRules := e.UnwrapBNF(c)
		newRules = mapsMerge(newRules, evenMoreRules)
		return evenExploded
	})

	res := []IdentSet{}
	for _, possible := range slices.Possibles(exploded) {
		res = append(res, slices.AppendMany(possible...))
	}

	return res, newRules
}

type Alts []Expr

var _ Expr = Alts{}

func (_ Alts) expr()          {}
func (a Alts) String() string { return stringify(a, " | ") }
func (a Alts) UnwrapBNF(c func() Ident) (res []IdentSet, newRules RuleSet) {
	newRules = make(RuleSet)

	for _, e := range a {
		evenExploded, evenMoreRules := e.UnwrapBNF(c)
		newRules = mapsMerge(newRules, evenMoreRules)
		res = append(res, evenExploded...)
	}

	return res, newRules
}

///
//////
/////////
////////////
/////////
//////
///

func stringify[S ~[]T, T fmt.Stringer](s S, sep string) string {
	return strings.Join(slices.Remap(s, func(_ int, v T) string { return v.String() }), sep)
}

func fmtStringify[S ~[]T, T any](s S, sep string) string {
	return strings.Join(slices.Remap(s, func(_ int, v T) string { return fmt.Sprint(v) }), sep)
}

func mapsMerge[M ~map[K]V, K comparable, V any](base M, maps ...M) M {
	if base == nil {
		base = make(M)
	}

	for _, item := range maps {
		for k, v := range item {
			base[k] = v
		}
	}
	return base
}

func mapsRemap[M1 ~map[K1]V1, K1, K2 comparable, V1, V2 any](m M1, f func(K1, V1) (K2, V2)) map[K2]V2 {
	res := make(map[K2]V2, len(m))
	for k1, v1 := range m {
		k2, v2 := f(k1, v1)
		res[k2] = v2
	}

	return res
}
