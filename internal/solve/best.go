package solve

import "github.com/Gulliby-development/narrowfit/internal/core"

// Best returns the first solution without an objective, or the best
// evaluable objective solution while preserving the first tie.
func Best(m *core.Model) (values []int, objective int64, ok bool) {
	if m == nil {
		return nil, 0, false
	}
	if m.Obj == nil {
		values, ok = First(m)
		return values, 0, ok
	}

	Enumerate(m.Vars, constraintsFromModel(m), func(candidate []int) bool {
		score, evaluable := core.Eval(m.Obj.E, candidate)
		if !evaluable {
			return true
		}
		if !ok || m.Obj.Maximize && score > objective || !m.Obj.Maximize && score < objective {
			values = candidate
			objective = score
			ok = true
		}
		return true
	})
	return values, objective, ok
}
