package grammar

import (
	"encoding/binary"
	"fmt"
	"strings"
	"unicode"

	"github.com/quenbyako/parser/constraints"
	"github.com/quenbyako/parser/slices"
	"github.com/zeebo/xxh3"
	"golang.org/x/exp/maps"
)

// TODO: remove it, but i don't know how...
const constIdentName = "CONST"

const emptyHash uint64 = 0x2d06800538d394c2

type IdentCounter map[string]uint64

func (c IdentCounter) NewIdent(id string) Ident {
	c[id]++
	return Ident{ID: id, AttrHash: c[id], Generated: true}
}

type Ident struct {
	ID       string
	AttrHash uint64

	Generated bool
}

func (_ Ident) expr() {}

var _ Expr = Ident{}

func (i Ident) String() string {
	if i.AttrHash == 0 || i.AttrHash == emptyHash && !i.Generated {
		return i.ID
	}

	if i.Generated {
		return fmt.Sprintf("%v_%d", i.ID, i.AttrHash)
	}

	return fmt.Sprintf("%v_%016x", i.ID, i.AttrHash)
}
func (i Ident) UnwrapBNF(func() Ident) ([]IdentSet, RuleSet) { return []IdentSet{{i}}, nil }
func (i Ident) Eq(k Ident) bool                              { return i.Cmp(k) == 0 }
func (i Ident) Cmp(k Ident) int {
	switch {
	case i.ID != k.ID:
		return constraints.Comparator(i.ID, k.ID)
	case i.AttrHash != k.AttrHash:
		return constraints.Comparator(i.AttrHash, k.AttrHash)
	default:
		return 0
	}
}

func (i Ident) Hash() (uint64, error) {
	res := []byte(i.ID)
	res = binary.LittleEndian.AppendUint64(res, i.AttrHash)
	if i.Generated {
		res = append(res, 1)
	} else {
		res = append(res, 0)
	}

	return xxh3.Hash(res), nil
}

type ComplexIdent struct {
	ID         string
	Properties map[string]*string
}

func (i ComplexIdent) String() string {
	if len(i.Properties) == 0 {
		return i.ID
	}
	return i.ID + "<" + i.metadata() + ">"
}

func (i ComplexIdent) Hash() (uint64, error) { return xxh3.HashString(i.metadata()), nil }

// в качестве аргумента подается (не)терминал, который нужно изучить
//
// объект, у которого вызывается метод является фильтром
func (i ComplexIdent) Select(o ComplexIdent) bool {
	if len(i.Properties) == 0 {
		return i.ID == o.ID
	}

	// guaranteed that i requires some attributes, so returning false
	if len(o.Properties) == 0 {
		return false
	}

	for k, v1 := range i.Properties {
		if v2, ok := o.Properties[k]; v1 == nil {
			return ok
		} else if v2 == nil || *v1 != *v2 {
			return false
		}
	}

	return true
}

func (i ComplexIdent) metadata() string {
	if len(i.Properties) == 0 {
		return ""
	}

	keys := slices.Sort(maps.Keys(i.Properties))
	metadata := make([]string, len(keys))
	for j, k := range keys {
		if v := i.Properties[k]; v != nil {
			metadata[j] = k + "=" + normalizeMetadataValue(*v)
		} else {
			metadata[j] = k
		}
	}

	return strings.Join(metadata, " ")
}

func normalizeMetadataValue(s string) string {
	if isIdent(s) {
		return "\"" + s + "\""
	}
	return s
}

func isIdent(s string) bool {
	var i int
	return !slices.ContainsFunc([]rune(s), func(r rune) bool {
		defer func() { i++ }()
		return !isIdentRune(r, i)
	})
}

func isIdentRune(r rune, i int) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) && i > 0
}

func typeIndex(funcs ...func(any) bool) func(any) int {
	return func(a any) int {
		return slices.IndexFunc(funcs, func(f func(any) bool) bool { return f(a) })
	}
}

func convertible[T any](i any) bool {
	_, ok := i.(T)
	return ok
}
