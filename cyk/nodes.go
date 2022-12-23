package cyk

import (
	"fmt"
	"strings"

	"github.com/quenbyako/parser/ebnf"
)

// Terminal это конечная возможная нода. Собственно, терминал.
type Terminal struct {
	ebnf.IdentNoEpsilon

	Obj fmt.Stringer
}

func (t Terminal) String() string            { return t.Obj.String() }
func (t Terminal) Name() ebnf.IdentNoEpsilon { return t.IdentNoEpsilon }

// RawNonTerminal это тот терминал, который генерирует алгоритм cyk, то есть буквально сырой терминал, который
// не сжат, не преобразован, и является оригинальным нетерминалом, который согласно алгоритму был сгенерирован
type RawNonTerminal struct {
	ebnf.IdentNoEpsilon

	Left  Node
	Right Node
}

func (t RawNonTerminal) String() string            { return t.Left.String() + t.Right.String() }
func (t RawNonTerminal) Name() ebnf.IdentNoEpsilon { return t.IdentNoEpsilon }

// NonTerminal это фактический нетерминал, который отдает в ответе парсер. в отличии от RawNonTerminal этот
// тип уже не предполагает наличия в дереве сгенерированных нетерминалов
type NonTerminal struct {
	ebnf.IdentNoEpsilon

	Nodes []Node
}

func (t NonTerminal) Name() ebnf.IdentNoEpsilon { return t.IdentNoEpsilon }
func (t NonTerminal) String() string {
	res := make([]string, len(t.Nodes))
	for i, node := range t.Nodes {
		res[i] = node.String()
	}

	return strings.Join(res, "")
}

// SingleNonTerminal это переходный нетерминал для корректной работы алгоритма. проблема в том, что существует
// некоторые грамматики, которые содержат в себе лишь один селектор. что бы не модифицировать алгоритм, в ту
// же самую ячейку записывается одиночный нетерминал, который записывается в ту же самую ячейку. это позволяет
// сохранить возможность наличия в нетерминале лишь одного элемента
type SingleNonTerminal struct {
	ebnf.IdentNoEpsilon

	Node Node
}

func (t SingleNonTerminal) String() string            { return t.Node.String() }
func (t SingleNonTerminal) Name() ebnf.IdentNoEpsilon { return t.IdentNoEpsilon }
