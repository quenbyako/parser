package grammar

import (
	"fmt"
	"io"

	"github.com/alecthomas/participle/v2"
	"golang.org/x/exp/maps"

	"github.com/quenbyako/parser/constraints"
	"github.com/quenbyako/parser/slices"
	"github.com/zeebo/xxh3"
)

// WTF??? https://github.com/golang/go/issues/46477
type Set[T comparable] map[T]struct{}

func (s Set[T]) Append(k T) Set[T] {
	if s == nil {
		s = make(Set[T])
	}
	s[k] = struct{}{}
	return s
}

func (s Set[T]) Has(k T) bool {
	_, ok := s[k]
	return ok
}

func (s Set[T]) String() string { return "set[" + fmtStringify(maps.Keys(s), " ") + "]" }

type HashSet[T Hasher] map[uint64]T

func (s HashSet[T]) Has(k T) bool {
	if s == nil {
		return false
	}

	h, err := k.Hash()
	if err != nil {
		panic(err)
	}

	_, ok := s[h]
	return ok
}

func (s HashSet[T]) Append(k ...T) HashSet[T] {
	if s == nil {
		s = make(HashSet[T])
	}

	for _, item := range k {
		h, err := item.Hash()
		if err != nil {
			panic(err)
		}

		s[h] = item
	}

	return s
}

func (s HashSet[T]) String() string { return "set[" + fmtStringify(maps.Values(s), " ") + "]" }

type hashCompare[T any] interface {
	Hasher
	constraints.Compare[T]
}

type Hasher interface {
	Hash() (uint64, error)
}

type grammar struct {
	P []production `parser:"@@*"`
}

func (g grammar) normalize(terms Set[string]) *EBNF {
	res := &EBNF{
		Rules: make(map[Ident][]Expr),

		Terminals: make(map[Ident]ComplexIdent),
		Constants: make(map[uint64]string),
	}
	for _, p := range g.P {
		name, expr := p.normalize(res, terms)
		exprs := []Expr{expr}
		if alts, ok := expr.(Alts); ok {
			exprs = alts
		}

		res.Rules[name] = append(res.Rules[name], exprs...)
	}

	return res
}

type production struct {
	C string `parser:"@Comment?"`
	N name   `parser:"@@ (':')"`
	E alts   `parser:"@@ ';'"`
}

func (p production) normalize(n *EBNF, terms Set[string]) (Ident, Expr) {
	name := p.N.asNonTerm()
	return name, p.E.normalize(n, terms)
}

type name struct {
	Ident  string          `parser:"@Ident"`
	Params []identMetadata `parser:"( '<' @@ + '>' )?"`
}

type identMetadata struct {
	Key   string  `parser:"@Ident"`
	Value *string `parser:"( '=' ( @Ident | @String ) )?"`
}

func (i name) normalize(n *EBNF, terms Set[string]) Expr {
	if _, ok := terms[i.Ident]; !ok {
		return i.asNonTerm()
	}

	return i.asTerm(n)
}

func (i name) asTerm(n *EBNF) Ident {
	params := make(map[string]*string, len(i.Params))
	for _, item := range i.Params {
		params[item.Key] = item.Value
	}

	replacer := ComplexIdent{
		ID:         i.Ident,
		Properties: params,
	}

	hash, _ := replacer.Hash()
	ident := Ident{ID: i.Ident, AttrHash: hash}
	n.Terminals[ident] = replacer

	return ident
}

func (i name) asNonTerm() Ident {
	if len(i.Params) == 0 {
		return Ident{ID: i.Ident}
	}

	params := make(map[string]*string, len(i.Params))
	for _, item := range i.Params {
		params[item.Key] = item.Value
	}

	panic(fmt.Sprintf("nonterminal %v not allowed to get params (for now)", ComplexIdent{i.Ident, params}))
}

type alts struct {
	A []sequence `parser:"@@ ( '|' @@ )*"`
}

func (e *alts) normalize(n *EBNF, terms Set[string]) Expr {
	if len(e.A) == 1 {
		return e.A[0].normalize(n, terms)
	}

	res := make(Alts, len(e.A))
	for i, alt := range e.A {
		res[i] = alt.normalize(n, terms)
	}

	return res
}

type sequence struct {
	T []term `parser:"@@+"`
}

func (s sequence) normalize(n *EBNF, terms Set[string]) Expr {
	if len(s.T) == 1 {
		return s.T[0].normalize(n, terms)
	}

	res := make(Seq, len(s.T))
	for i, term := range s.T {
		res[i] = term.normalize(n, terms)
	}

	return res
}

type group struct {
	E alts `parser:"'(' @@ ')'"`
}

func (g group) normalize(n *EBNF, terms Set[string]) Expr {
	if len(g.E.A) == 1 {
		return g.E.A[0].normalize(n, terms)
	}

	return Group{g.E.normalize(n, terms)}
}

type option struct {
	E alts `parser:"'[' @@ ']'"`
}

func (o option) normalize(n *EBNF, terms Set[string]) Expr {
	return Option{E: o.E.normalize(n, terms)}
}

type repeat struct {
	E alts `parser:"'{' @@ '}'"`
}

func (r repeat) normalize(n *EBNF, terms Set[string]) Expr {
	return Repeat{E: r.E.normalize(n, terms)}
}

type term struct {
	Const  *string `parser:"@String |"`
	Name   *name   `parser:"@@ |"`
	Group  *group  `parser:"@@ |"`
	Option *option `parser:"@@ |"`
	Repeat *repeat `parser:"@@"`
}

func (t term) normalize(n *EBNF, terms Set[string]) Expr {
	switch {
	case t.Const != nil:
		hash := xxh3.HashString(*t.Const)
		n.Constants[hash] = *t.Const

		return Ident{ID: constIdentName, AttrHash: hash}
	case t.Name != nil:
		return t.Name.normalize(n, terms)
	case t.Group != nil:
		return t.Group.normalize(n, terms)
	case t.Option != nil:
		return t.Option.normalize(n, terms)
	case t.Repeat != nil:
		return t.Repeat.normalize(n, terms)
	default:
		panic("wut")
	}
}

var parser = participle.MustBuild[grammar](
	participle.Unquote("String"),
)

// грамматика уже очищенна от терминалов и констант.
//
// например если вы передадите терминал-селектор без аттрибутов ( noun<> ), то в
// самой грамматике он будет заменен на:
//
//	parser.BaseIdent{
//	    ID:        "noun",
//	    Hash:      0x2d06800538d394c2,
//	}
//
// терминалы-селекторы без аттрибутов будут заменены идентичным пустым хешем
// (для xxh3 это 2d06800538d394c2)
func Parse(file string, input io.Reader, terminals ...string) (*EBNF, error) {
	g, err := parser.Parse(file, input)
	if err != nil {
		return nil, err
	}

	return g.normalize(slices.ToMap(terminals)), nil
}
