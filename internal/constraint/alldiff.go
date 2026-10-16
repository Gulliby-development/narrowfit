package constraint

import (
	"sort"
	"strings"

	"github.com/Gulliby-development/narrowfit/internal/core"
)

type allDifferent struct {
	scope []int
	text  string
}

func buildAlldiff(words []string, vars []core.Var) (core.Constraint, string) {
	if len(words) < 2 {
		return nil, "invalid alldiff statement"
	}

	indexes, msg := core.ResolveNames(vars, words)
	if msg != "" {
		return nil, msg
	}
	sort.Ints(indexes)

	var text strings.Builder
	text.WriteString("alldiff")
	for _, word := range words {
		text.WriteByte(' ')
		text.WriteString(word)
	}

	return &allDifferent{scope: indexes, text: text.String()}, ""
}

func (c *allDifferent) Scope() []int {
	return append([]int(nil), c.scope...)
}

func (c *allDifferent) Holds(values []int) bool {
	seen := make(map[int]struct{}, len(c.scope))
	for _, index := range c.scope {
		if index < 0 || index >= len(values) {
			return false
		}
		if _, ok := seen[values[index]]; ok {
			return false
		}
		seen[values[index]] = struct{}{}
	}
	return true
}

func (c *allDifferent) Text() string {
	return c.text
}
