package constraint

import (
	"strings"

	"github.com/Gulliby-development/narrowfit/internal/core"
)

type elementConstraint struct {
	index  int
	middle []int
	result int
	scope  []int
	text   string
}

func buildElement(words []string, vars []core.Var) (core.Constraint, string) {
	if len(words) < 5 || words[1] != "in" || words[len(words)-2] != "is" {
		return nil, "invalid element statement"
	}

	middleNames := words[2 : len(words)-2]
	indexName := words[0]
	resultName := words[len(words)-1]

	names := make([]string, 0, len(middleNames)+2)
	names = append(names, indexName)
	names = append(names, middleNames...)
	names = append(names, resultName)

	indexes := make([]int, len(names))
	for i, name := range names {
		index, ok := core.VarIndex(vars, name)
		if !ok {
			return nil, "unknown variable \"" + name + "\""
		}
		indexes[i] = index
	}

	middleIndexes, msg := core.ResolveNames(vars, middleNames)
	if msg != "" {
		return nil, msg
	}

	referenced := make([]bool, len(vars))
	referenced[indexes[0]] = true
	for _, index := range middleIndexes {
		referenced[index] = true
	}
	resultIndex := indexes[len(indexes)-1]
	referenced[resultIndex] = true

	scope := make([]int, 0, len(vars))
	for index, used := range referenced {
		if used {
			scope = append(scope, index)
		}
	}

	text := "element " + indexName + " in " + strings.Join(middleNames, " ") + " is " + resultName
	return &elementConstraint{
		index:  indexes[0],
		middle: middleIndexes,
		result: resultIndex,
		scope:  scope,
		text:   text,
	}, ""
}

func (c *elementConstraint) Scope() []int {
	return c.scope
}

func (c *elementConstraint) Holds(values []int) bool {
	if c.index < 0 || c.index >= len(values) {
		return false
	}
	position := values[c.index]
	if position < 1 || position > len(c.middle) {
		return false
	}

	selected := c.middle[position-1]
	if selected < 0 || selected >= len(values) || c.result < 0 || c.result >= len(values) {
		return false
	}
	return values[selected] == values[c.result]
}

func (c *elementConstraint) Text() string {
	return c.text
}
