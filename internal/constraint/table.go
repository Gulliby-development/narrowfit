package constraint

import (
	"sort"
	"strings"

	"github.com/Gulliby-development/narrowfit/internal/core"
)

const invalidTableStatement = "invalid table statement"

type tableConstraint struct {
	indexes []int
	scope   []int
	tuples  [][]int
	text    string
}

func buildTable(words []string, vars []core.Var) (core.Constraint, string) {
	colon := -1
	for i, word := range words {
		if word == ":" {
			if colon >= 0 {
				return nil, invalidTableStatement
			}
			colon = i
		}
	}
	if colon <= 0 || colon == len(words)-1 {
		return nil, invalidTableStatement
	}

	names := words[:colon]
	for _, name := range names {
		if name == "" || name == "|" {
			return nil, invalidTableStatement
		}
	}

	tuples := make([][]int, 0, 1)
	tuple := make([]int, 0, len(names))
	for _, word := range words[colon+1:] {
		switch word {
		case ":", "":
			return nil, invalidTableStatement
		case "|":
			if len(tuple) != len(names) {
				return nil, invalidTableStatement
			}
			tuples = append(tuples, tuple)
			tuple = make([]int, 0, len(names))
		default:
			value, ok := core.ParseInt(word, -1000, 1000)
			if !ok {
				return nil, invalidTableStatement
			}
			tuple = append(tuple, value)
			if len(tuple) > len(names) {
				return nil, invalidTableStatement
			}
		}
	}
	if len(tuple) != len(names) {
		return nil, invalidTableStatement
	}
	tuples = append(tuples, tuple)

	indexes, msg := core.ResolveNames(vars, names)
	if msg != "" {
		return nil, msg
	}

	scope := append([]int(nil), indexes...)
	sort.Ints(scope)

	return &tableConstraint{
		indexes: indexes,
		scope:   scope,
		tuples:  tuples,
		text:    "table " + strings.Join(words, " "),
	}, ""
}

func (c *tableConstraint) Scope() []int {
	return append([]int(nil), c.scope...)
}

func (c *tableConstraint) Holds(values []int) bool {
	for _, tuple := range c.tuples {
		matches := true
		for column, index := range c.indexes {
			if index < 0 || index >= len(values) || values[index] != tuple[column] {
				matches = false
				break
			}
		}
		if matches {
			return true
		}
	}
	return false
}

func (c *tableConstraint) Text() string {
	return c.text
}
