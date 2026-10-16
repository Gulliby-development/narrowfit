// Package solve provides deterministic search over core model assignments.
package solve

import "github.com/Gulliby-development/narrowfit/internal/core"

// Enumerate calls visit with each satisfying assignment in declaration and
// domain order. It stops immediately when visit returns false.
func Enumerate(vars []core.Var, cons []core.Constraint, visit func(values []int) bool) {
	if visit == nil {
		return
	}

	values := make([]int, len(vars))
	ready := make([][]core.Constraint, len(vars))
	for _, constraint := range cons {
		scope := constraint.Scope()
		if len(scope) == 0 {
			if !constraint.Holds(values) {
				return
			}
			continue
		}
		last := scope[len(scope)-1]
		ready[last] = append(ready[last], constraint)
	}

	var descend func(index int) bool
	descend = func(index int) bool {
		if index == len(vars) {
			return visit(append([]int(nil), values...))
		}

		for _, value := range vars[index].Values {
			values[index] = value
			satisfied := true
			for _, constraint := range ready[index] {
				if !constraint.Holds(values) {
					satisfied = false
					break
				}
			}
			if satisfied && !descend(index+1) {
				return false
			}
		}
		return true
	}

	descend(0)
}

// First returns the first satisfying assignment, ignoring any objective.
func First(m *core.Model) ([]int, bool) {
	if m == nil {
		return nil, false
	}

	var first []int
	found := false
	Enumerate(m.Vars, constraintsFromModel(m), func(values []int) bool {
		first = values
		found = true
		return false
	})
	return first, found
}

// Count visits up to limit satisfying assignments and reports whether another
// solution exists. The extra solution used to determine more is never visited.
func Count(m *core.Model, limit int, visit func(values []int)) (n int, more bool) {
	if m == nil {
		return 0, false
	}

	if limit <= 0 {
		Enumerate(m.Vars, constraintsFromModel(m), func([]int) bool {
			more = true
			return false
		})
		return 0, more
	}

	Enumerate(m.Vars, constraintsFromModel(m), func(values []int) bool {
		if n == limit {
			more = true
			return false
		}
		n++
		if visit != nil {
			visit(values)
		}
		return true
	})
	return n, more
}

// constraintsFromModel adapts the model's statement list for search APIs.
func constraintsFromModel(m *core.Model) []core.Constraint {
	if m == nil {
		return nil
	}
	constraints := make([]core.Constraint, len(m.Stmts))
	for i, statement := range m.Stmts {
		constraints[i] = statement.C
	}
	return constraints
}
