package solve_test

import (
	"reflect"
	"testing"

	"github.com/Gulliby-development/narrowfit/internal/core"
	"github.com/Gulliby-development/narrowfit/internal/solve"
)

var _ func(*core.Model) ([]int, bool) = solve.Explain

type explainConstraint struct {
	scope []int
	text  string
	holds func([]int) bool
}

func (c *explainConstraint) Scope() []int { return c.scope }

func (c *explainConstraint) Holds(values []int) bool {
	return c.holds == nil || c.holds(values)
}

func (c *explainConstraint) Text() string { return c.text }

func explainVar(name string, values []int) core.Var {
	return core.Var{Name: name, Values: values}
}

func explainPredicate(scope []int, text string, holds func([]int) bool) *explainConstraint {
	return &explainConstraint{scope: scope, text: text, holds: holds}
}

func explainEqual(index, want int, text string) *explainConstraint {
	return explainPredicate([]int{index}, text, func(values []int) bool {
		return values[index] == want
	})
}

func explainModel(vars []core.Var, constraints ...core.Constraint) *core.Model {
	statements := make([]core.Stmt, len(constraints))
	for i, constraint := range constraints {
		statements[i] = core.Stmt{Line: i + 1, C: constraint}
	}
	return &core.Model{Vars: vars, Stmts: statements}
}

func explainSubset(m *core.Model, indexes []int) *core.Model {
	statements := make([]core.Stmt, 0, len(indexes))
	for _, index := range indexes {
		statements = append(statements, m.Stmts[index])
	}
	return &core.Model{Vars: m.Vars, Stmts: statements, Obj: m.Obj, Text: m.Text}
}

func TestExplainDeterministicConflictCases(t *testing.T) {
	x := []core.Var{explainVar("x", []int{1, 2})}
	cases := []struct {
		name            string
		model           *core.Model
		wantConflict    []int
		wantSatisfiable bool
	}{
		{
			name:            "unconstrained model with unevaluable objective is satisfiable",
			model:           &core.Model{Vars: x, Obj: &core.Objective{E: core.Bin{Op: "/", L: core.Num{V: 1}, R: core.Num{V: 0}}}},
			wantSatisfiable: true,
		},
		{
			name: "constrained model is satisfiable",
			model: explainModel(x,
				explainEqual(0, 2, "x == 2"),
			),
			wantSatisfiable: true,
		},
		{
			name: "zero variables and true empty scope are satisfiable",
			model: explainModel(nil,
				explainPredicate(nil, "require 1 < 2", func([]int) bool { return true }),
			),
			wantSatisfiable: true,
		},
		{
			name:            "zero variables and false empty scope form a singleton conflict",
			model:           explainModel(nil, explainPredicate(nil, "require 1 > 2", func([]int) bool { return false })),
			wantConflict:    []int{0},
			wantSatisfiable: false,
		},
		{
			name:            "no constraints and zero variables is satisfiable",
			model:           &core.Model{},
			wantSatisfiable: true,
		},
		{
			name: "duplicate false constraints retain the later original statement",
			model: explainModel(x,
				explainPredicate(nil, "require false a", func([]int) bool { return false }),
				explainPredicate(nil, "require false b", func([]int) bool { return false }),
			),
			wantConflict: []int{1},
		},
		{
			name: "redundant contradictory constraint is deleted in statement order",
			model: explainModel(x,
				explainEqual(0, 1, "x == 1 first"),
				explainEqual(0, 2, "x == 2"),
				explainEqual(0, 1, "x == 1 redundant"),
			),
			wantConflict: []int{1, 2},
		},
		{
			name: "multiple conflicts deterministically retain the later pair",
			model: explainModel(
				[]core.Var{x[0], explainVar("y", []int{1, 2})},
				explainEqual(0, 1, "x == 1"),
				explainEqual(0, 2, "x == 2"),
				explainEqual(1, 1, "y == 1"),
				explainEqual(1, 2, "y == 2"),
			),
			wantConflict: []int{2, 3},
		},
		{
			name: "returned values are statement indexes not source lines",
			model: &core.Model{
				Vars: x,
				Stmts: []core.Stmt{
					{Line: 2, C: explainEqual(0, 1, "x == 1")},
					{Line: 40, C: explainPredicate(nil, "require true", func([]int) bool { return true })},
					{Line: 90, C: explainEqual(0, 2, "x == 2")},
				},
			},
			wantConflict: []int{0, 2},
		},
		{
			name: "false empty scope with another constraint remains a singleton",
			model: explainModel(x,
				explainPredicate(nil, "require false", func([]int) bool { return false }),
				explainEqual(0, 1, "x == 1"),
			),
			wantConflict: []int{0},
		},
		{
			name: "rooms comparisons are the exact conflict after alldiff is dropped",
			model: explainModel(
				[]core.Var{
					explainVar("ann", []int{1, 2, 3}),
					explainVar("bob", []int{1, 2, 3}),
					explainVar("cy", []int{1, 2, 3}),
				},
				explainPredicate([]int{0, 1, 2}, "alldiff ann bob cy", func(values []int) bool {
					return values[0] != values[1] && values[0] != values[2] && values[1] != values[2]
				}),
				explainPredicate([]int{0, 1}, "ann < bob", func(values []int) bool { return values[0] < values[1] }),
				explainPredicate([]int{1, 2}, "bob < cy", func(values []int) bool { return values[1] < values[2] }),
				explainPredicate([]int{0, 2}, "cy < ann", func(values []int) bool { return values[2] < values[0] }),
			),
			wantConflict: []int{1, 2, 3},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			conflict, satisfiable := solve.Explain(tc.model)
			if satisfiable != tc.wantSatisfiable || !reflect.DeepEqual(conflict, tc.wantConflict) {
				t.Fatalf("Explain() = (%v, %v), want (%v, %v)", conflict, satisfiable, tc.wantConflict, tc.wantSatisfiable)
			}

			if satisfiable {
				if len(conflict) != 0 {
					t.Fatalf("satisfiable model returned conflict %v", conflict)
				}
				return
			}

			if _, ok := solve.First(explainSubset(tc.model, conflict)); ok {
				t.Fatalf("returned subset %v is satisfiable", conflict)
			}
			for _, removed := range conflict {
				remaining := make([]int, 0, len(conflict)-1)
				for _, index := range conflict {
					if index != removed {
						remaining = append(remaining, index)
					}
				}
				if _, ok := solve.First(explainSubset(tc.model, remaining)); !ok {
					t.Fatalf("removing statement index %d from conflict %v did not restore satisfiability", removed, conflict)
				}
			}
		})
	}
}

func TestExplainRoomsResultIsRepeatable(t *testing.T) {
	model := explainModel(
		[]core.Var{
			explainVar("ann", []int{1, 2, 3}),
			explainVar("bob", []int{1, 2, 3}),
			explainVar("cy", []int{1, 2, 3}),
		},
		explainPredicate([]int{0, 1, 2}, "alldiff ann bob cy", func(values []int) bool {
			return values[0] != values[1] && values[0] != values[2] && values[1] != values[2]
		}),
		explainPredicate([]int{0, 1}, "ann < bob", func(values []int) bool { return values[0] < values[1] }),
		explainPredicate([]int{1, 2}, "bob < cy", func(values []int) bool { return values[1] < values[2] }),
		explainPredicate([]int{0, 2}, "cy < ann", func(values []int) bool { return values[2] < values[0] }),
	)

	for attempt := 0; attempt < 3; attempt++ {
		conflict, satisfiable := solve.Explain(model)
		if satisfiable || !reflect.DeepEqual(conflict, []int{1, 2, 3}) {
			t.Fatalf("Explain() attempt %d = (%v, %v), want ([1 2 3], false)", attempt+1, conflict, satisfiable)
		}
	}
}

func TestExplainDoesNotMutateModelOrBackingData(t *testing.T) {
	values := make([]int, 2, 3)
	values[0], values[1] = 1, 2
	values[:cap(values)][2] = 77
	vars := make([]core.Var, 1, 2)
	vars[0] = core.Var{Name: "x", Values: values}
	vars[:cap(vars)][1] = core.Var{Name: "hidden", Values: []int{8, 9}}

	constraintA := explainEqual(0, 1, "x == 1")
	constraintB := explainEqual(0, 2, "x == 2")
	statements := make([]core.Stmt, 2, 3)
	statements[0] = core.Stmt{Line: 3, C: constraintA}
	statements[1] = core.Stmt{Line: 88, C: constraintB}
	statements[:cap(statements)][2] = core.Stmt{Line: 101, C: explainPredicate(nil, "hidden", nil)}

	text := make([]string, 2, 3)
	text[0], text[1] = "var x 1..2", "objective"
	text[:cap(text)][2] = "backing sentinel"
	objective := &core.Objective{Line: 99, Maximize: true, E: core.Ref{Index: 0, Name: "x"}}
	model := &core.Model{Vars: vars, Stmts: statements, Obj: objective, Text: text}

	wantVars := append([]core.Var(nil), vars[:cap(vars)]...)
	wantValues := append([]int(nil), values[:cap(values)]...)
	wantStatements := append([]core.Stmt(nil), statements[:cap(statements)]...)
	wantText := append([]string(nil), text[:cap(text)]...)
	wantObjective := *objective
	wantScopeA := append([]int(nil), constraintA.scope...)
	wantScopeB := append([]int(nil), constraintB.scope...)

	conflict, satisfiable := solve.Explain(model)
	if satisfiable || !reflect.DeepEqual(conflict, []int{0, 1}) {
		t.Fatalf("Explain() = (%v, %v), want ([0 1], false)", conflict, satisfiable)
	}
	if !reflect.DeepEqual(model.Vars[:cap(model.Vars)], wantVars) {
		t.Fatalf("Vars or its backing data changed: got %v, want %v", model.Vars[:cap(model.Vars)], wantVars)
	}
	if !reflect.DeepEqual(model.Vars[0].Values[:cap(model.Vars[0].Values)], wantValues) {
		t.Fatalf("domain or its backing data changed: got %v, want %v", model.Vars[0].Values[:cap(model.Vars[0].Values)], wantValues)
	}
	if !reflect.DeepEqual(model.Stmts[:cap(model.Stmts)], wantStatements) {
		t.Fatalf("Stmts or its backing data changed: got %v, want %v", model.Stmts[:cap(model.Stmts)], wantStatements)
	}
	if !reflect.DeepEqual(model.Text[:cap(model.Text)], wantText) {
		t.Fatalf("Text or its backing data changed: got %v, want %v", model.Text[:cap(model.Text)], wantText)
	}
	if model.Obj != objective || !reflect.DeepEqual(*model.Obj, wantObjective) {
		t.Fatalf("Obj changed: got %v, want %v", model.Obj, &wantObjective)
	}
	if !reflect.DeepEqual(constraintA.scope, wantScopeA) || !reflect.DeepEqual(constraintB.scope, wantScopeB) {
		t.Fatalf("constraint scope changed: %v and %v", constraintA.scope, constraintB.scope)
	}
}
