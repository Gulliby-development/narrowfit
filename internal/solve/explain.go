package solve

import "github.com/Gulliby-development/narrowfit/internal/core"

// Explain returns a deterministic inclusion-minimal unsatisfiable subset of
// constraint statement indexes, or reports that the model is satisfiable.
func Explain(m *core.Model) (conflict []int, satisfiable bool) {
	if m == nil {
		return nil, false
	}
	if _, ok := First(m); ok {
		return nil, true
	}

	active := make([]bool, len(m.Stmts))
	for i := range active {
		active[i] = true
	}

	for candidate := range m.Stmts {
		if !active[candidate] {
			continue
		}

		trial := *m
		trial.Stmts = make([]core.Stmt, 0, len(m.Stmts)-1)
		for i, statement := range m.Stmts {
			if active[i] && i != candidate {
				trial.Stmts = append(trial.Stmts, statement)
			}
		}
		if _, ok := First(&trial); !ok {
			active[candidate] = false
		}
	}

	conflict = make([]int, 0, len(m.Stmts))
	for i, keep := range active {
		if keep {
			conflict = append(conflict, i)
		}
	}
	return conflict, false
}
