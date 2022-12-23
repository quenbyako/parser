package grammar

import (
	"encoding/binary"

	"github.com/quenbyako/parser/constraints"
	"github.com/quenbyako/parser/slices"
	"github.com/zeebo/xxh3"
)

type IdentSet []Ident

func (s IdentSet) String() string {
	if len(s) == 0 {
		return epsilonSymbol
	}

	return stringify(s, " ")
}

func (r IdentSet) Cmp(j IdentSet) int {
	for i := 0; i < len(r) && i < len(j); i++ {
		if res := r[i].Cmp(j[i]); res != 0 {
			return res
		}
	}

	return constraints.Comparator(len(r), len(j))
}

func (r IdentSet) Eq(j IdentSet) bool            { return slices.Equal(r, j) }
func (r IdentSet) isChain(terms Set[Ident]) bool { return len(r) == 1 && !terms.Has(r[0]) }

func (s IdentSet) Hash() (uint64, error) {
	if len(s) == 0 {
		return emptyHash, nil
	}

	res := make([]byte, 0, len(s)*8)
	for _, ident := range s {
		h, _ := ident.Hash()
		res = binary.LittleEndian.AppendUint64(res, h)
	}

	return xxh3.Hash(res), nil
}

type DualRule [2]Ident

func (r DualRule) Eq(j DualRule) bool { return r[0].Eq(j[0]) && r[1].Eq(j[1]) }

func (r DualRule) Cmp(j DualRule) int {
	if res := r[0].Cmp(j[0]); res != 0 {
		return res
	}

	return r[1].Cmp(j[1])
}

func (r DualRule) Hash() (uint64, error) {
	const uint64Size = 8
	res := make([]byte, uint64Size*2)
	h0, _ := r[0].Hash()
	binary.LittleEndian.PutUint64(res[0:8], h0)
	h1, _ := r[1].Hash()
	binary.LittleEndian.PutUint64(res[8:16], h1)

	return xxh3.Hash(res), nil
}
