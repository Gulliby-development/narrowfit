package constraint

import "github.com/Gulliby-development/narrowfit/internal/core"

// Build dispatches a word-based constraint statement to its builder.
func Build(keyword string, words []string, vars []core.Var) (c core.Constraint, msg string, known bool) {
	switch keyword {
	case "alldiff":
		c, msg = buildAlldiff(words, vars)
	case "sum":
		c, msg = buildSum(words, vars)
	case "count":
		c, msg = buildCount(words, vars)
	case "table":
		c, msg = buildTable(words, vars)
	case "element":
		c, msg = buildElement(words, vars)
	default:
		return nil, "", false
	}
	return c, msg, true
}
