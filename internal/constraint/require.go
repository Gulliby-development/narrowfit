// Package constraint provides model constraints built on core expressions and types.
package constraint

import "github.com/Gulliby-development/narrowfit/internal/core"

type require struct {
	expr core.Expr
}

// NewRequire returns a constraint requiring e to evaluate to true.
func NewRequire(e core.Expr) core.Constraint {
	return require{expr: e}
}

func (r require) Scope() []int {
	return core.ExprVars(r.expr)
}

func (r require) Holds(values []int) bool {
	value, ok := core.Eval(r.expr, values)
	return ok && value == 1
}

func (r require) Text() string {
	return "require " + core.ExprText(r.expr)
}
