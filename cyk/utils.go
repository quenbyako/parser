package cyk

import (
	"fmt"
)

// xy is a alias to 2 dimension coordinates for maps inside parser matrix
type xy struct{ x, y uint }

func (n xy) String() string { return fmt.Sprintf("coords[%v:%v]", n.x, n.y) }
