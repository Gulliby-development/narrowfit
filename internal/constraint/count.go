package constraint

import (
	"sort"
	"strconv"
	"strings"

	"github.com/Gulliby-development/narrowfit/internal/core"
)

type countConstraint struct {
	target    int
	threshold int
	op        string
	indexes   []int
	text      string
}

func buildCount(words []string, vars []core.Var) (core.Constraint, string) {
	if len(words) < 4 {
		return nil, "invalid count statement"
	}

	target, targetOK := core.ParseInt(words[0], -1000, 1000)
	threshold, thresholdOK := core.ParseInt(words[len(words)-1], 0, 1000)
	op := words[len(words)-2]
	if !targetOK || !thresholdOK || !validCountOperator(op) {
		return nil, "invalid count statement"
	}

	names := words[1 : len(words)-2]
	indexes, msg := core.ResolveNames(vars, names)
	if msg != "" {
		return nil, msg
	}

	scope := append([]int(nil), indexes...)
	sort.Ints(scope)
	text := "count " + strconv.Itoa(target) + " " + strings.Join(names, " ") +
		" " + op + " " + strconv.Itoa(threshold)

	return countConstraint{
		target:    target,
		threshold: threshold,
		op:        op,
		indexes:   scope,
		text:      text,
	}, ""
}

func validCountOperator(op string) bool {
	switch op {
	case "==", "!=", "<", "<=", ">", ">=":
		return true
	default:
		return false
	}
}

func (c countConstraint) Scope() []int {
	return append([]int(nil), c.indexes...)
}

func (c countConstraint) Holds(values []int) bool {
	var matches int64
	for _, index := range c.indexes {
		if values[index] == c.target {
			matches++
		}
	}
	return core.Compare(c.op, matches, int64(c.threshold))
}

func (c countConstraint) Text() string {
	return c.text
}
