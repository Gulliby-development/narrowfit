package constraint

import (
	"sort"
	"strings"

	"github.com/Gulliby-development/narrowfit/internal/core"
)

type sumConstraint struct {
	indexes []int
	op      string
	target  int64
	text    string
}

func buildSum(words []string, vars []core.Var) (core.Constraint, string) {
	if len(words) < 3 {
		return nil, "invalid sum statement"
	}

	op := words[len(words)-2]
	switch op {
	case "==", "!=", "<", "<=", ">", ">=":
	default:
		return nil, "invalid sum statement"
	}

	target, ok := core.ParseInt(words[len(words)-1], -1000000, 1000000)
	if !ok {
		return nil, "invalid sum statement"
	}

	names := words[:len(words)-2]
	indexes, msg := core.ResolveNames(vars, names)
	if msg != "" {
		return nil, msg
	}
	scope := append([]int(nil), indexes...)
	sort.Ints(scope)

	return &sumConstraint{
		indexes: scope,
		op:      op,
		target:  int64(target),
		text:    "sum " + strings.Join(words, " "),
	}, ""
}

func (c *sumConstraint) Scope() []int {
	return append([]int(nil), c.indexes...)
}

func (c *sumConstraint) Holds(values []int) bool {
	var total int64
	for _, index := range c.indexes {
		total += int64(values[index])
	}
	return core.Compare(c.op, total, c.target)
}

func (c *sumConstraint) Text() string {
	return c.text
}
