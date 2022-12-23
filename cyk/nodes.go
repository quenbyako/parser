package cyk

import (
	"text/scanner"

	"github.com/quenbyako/parser/grammar"
)

// Terminal это конечная нода. Собственно, терминал.
type Terminal struct {
	scanner.Position
	Type  grammar.Ident // ident name
	Value string
}

// NonTerminal это тот терминал, который генерирует алгоритм cyk, то есть
// буквально сырой терминал, который не сжат, не преобразован, и является
// оригинальным нетерминалом, который согласно алгоритму был сгенерирован
type NonTerminal struct {
	I grammar.Ident

	// ВАЖНО: если вы пытаетесь дернуть кординаты у нетерминала, который
	// находится в диагональной ячейке (где координаты x==y) то координаты
	// будут пустыми. остальные нетерминалы обязаны иметь координаты
	Left   NonTerminalCoord
	Bottom NonTerminalCoord
}


type NonTerminalCoord struct {
	XY
	Index int
}
