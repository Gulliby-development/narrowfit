package solve_test

import (
	"reflect"
	"testing"

	"github.com/Gulliby-development/narrowfit/internal/core"
	"github.com/Gulliby-development/narrowfit/internal/solve"
)

var (
	_ func([]core.Var, []core.Constraint, func([]int) bool) = solve.Enumerate
	_ func(*core.Model) ([]int, bool)                       = solve.First
	_ func(*core.Model) ([]int, int64, bool)                = solve.Best
	_ func(*core.Model, int, func([]int)) (int, bool)       = solve.Count
)

type searchConstraint struct {
	scope    []int
	text     string
	holds    func([]int) bool
	calls    int
	observed [][]int
}

func (c *searchConstraint) Scope() []int { return c.scope }

func (c *searchConstraint) Holds(values []int) bool {
	c.calls++
	c.observed = append(c.observed, append([]int(nil), values...))
	return c.holds == nil || c.holds(values)
}

func (c *searchConstraint) Text() string { return c.text }

func searchVar(name string, values []int) core.Var {
	return core.Var{Name: name, Values: values}
}

func searchModel(vars []core.Var, constraints []core.Constraint, objective *core.Objective) *core.Model {
	statements := make([]core.Stmt, len(constraints))
	for i, constraint := range constraints {
		statements[i] = core.Stmt{Line: i + 1, C: constraint}
	}
	return &core.Model{Vars: vars, Stmts: statements, Obj: objective}
}

func searchPredicate(scope []int, text string, holds func([]int) bool) *searchConstraint {
	return &searchConstraint{scope: scope, text: text, holds: holds}
}

func searchSum(target int, indexes ...int) *searchConstraint {
	return searchPredicate(indexes, "sum", func(values []int) bool {
		sum := 0
		for _, index := range indexes {
			sum += values[index]
		}
		return sum == target
	})
}

func searchAllDifferent(indexes ...int) *searchConstraint {
	return searchPredicate(indexes, "alldiff", func(values []int) bool {
		seen := make(map[int]bool, len(indexes))
		for _, index := range indexes {
			if seen[values[index]] {
				return false
			}
			seen[values[index]] = true
		}
		return true
	})
}

func TestEnumerateOrderedSearchCases(t *testing.T) {
	cases := []struct {
		name        string
		vars        []core.Var
		constraints []core.Constraint
		want        [][]int
	}{
		{
			name: "cartesian product preserves variable and domain order",
			vars: []core.Var{searchVar("x", []int{-1, 2}), searchVar("y", []int{0, 3})},
			want: [][]int{{-1, 0}, {-1, 3}, {2, 0}, {2, 3}},
		},
		{
			name: "single-scope filter preserves surviving order",
			vars: []core.Var{searchVar("x", []int{1, 2, 3})},
			constraints: []core.Constraint{
				searchPredicate([]int{0}, "even", func(values []int) bool { return values[0]%2 == 0 }),
			},
			want: [][]int{{2}},
		},
		{
			name: "empty-scope true constraint permits every assignment",
			vars: []core.Var{searchVar("x", []int{1, 2}), searchVar("y", []int{3, 4})},
			constraints: []core.Constraint{
				searchPredicate(nil, "true", func([]int) bool { return true }),
			},
			want: [][]int{{1, 3}, {1, 4}, {2, 3}, {2, 4}},
		},
		{
			name: "zero variables produce one empty assignment",
			want: [][]int{nil},
		},
		{
			name: "scope constraint can reject every assignment",
			vars: []core.Var{searchVar("x", []int{1, 2})},
			constraints: []core.Constraint{
				searchPredicate([]int{0}, "false", func([]int) bool { return false }),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got [][]int
			solve.Enumerate(tc.vars, tc.constraints, func(values []int) bool {
				got = append(got, values)
				return true
			})
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("Enumerate() = %v, want %v", got, tc.want)
			}
		})
	}

	t.Run("empty-scope false is checked once before descent", func(t *testing.T) {
		empty := searchPredicate(nil, "false", func([]int) bool { return false })
		later := searchPredicate([]int{0}, "later", func([]int) bool { return true })
		solve.Enumerate([]core.Var{searchVar("x", []int{1, 2})}, []core.Constraint{empty, later}, func([]int) bool {
			t.Fatal("visited after an empty-scope constraint failed")
			return true
		})
		if empty.calls != 1 || later.calls != 0 {
			t.Fatalf("Holds calls = (%d, %d), want (1, 0)", empty.calls, later.calls)
		}
	})

	t.Run("empty-scope true is checked once", func(t *testing.T) {
		empty := searchPredicate(nil, "true", nil)
		solve.Enumerate([]core.Var{searchVar("x", []int{1, 2, 3})}, []core.Constraint{empty}, func([]int) bool {
			return true
		})
		if empty.calls != 1 {
			t.Fatalf("empty-scope Holds calls = %d, want 1", empty.calls)
		}
	})

	t.Run("nil visitor does not start search", func(t *testing.T) {
		constraint := searchPredicate([]int{0}, "count calls", nil)
		solve.Enumerate([]core.Var{searchVar("x", []int{1, 2})}, []core.Constraint{constraint}, nil)
		if constraint.calls != 0 {
			t.Fatalf("Holds calls = %d, want 0", constraint.calls)
		}
	})
}

func TestEnumerateReadinessAndPruning(t *testing.T) {
	t.Run("waits for the last scoped variable and passes full assignments", func(t *testing.T) {
		constraint := searchPredicate([]int{0, 2}, "endpoints differ", func(values []int) bool {
			return values[0] != values[2]
		})
		var got [][]int
		solve.Enumerate(
			[]core.Var{searchVar("x", []int{1, 2}), searchVar("y", []int{8, 9}), searchVar("z", []int{1, 3})},
			[]core.Constraint{constraint},
			func(values []int) bool {
				got = append(got, values)
				return true
			},
		)
		want := [][]int{{1, 8, 3}, {1, 9, 3}, {2, 8, 1}, {2, 8, 3}, {2, 9, 1}, {2, 9, 3}}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("assignments = %v, want %v", got, want)
		}
		if constraint.calls != 8 {
			t.Fatalf("Holds calls = %d, want 8 (one for each x/y prefix)", constraint.calls)
		}
		for i, values := range constraint.observed {
			if len(values) != 3 || values[2] != 1 && values[2] != 3 {
				t.Fatalf("Holds call %d received %v, want a full assignment", i, values)
			}
		}
	})

	t.Run("rejects a failing prefix before descending", func(t *testing.T) {
		constraint := searchPredicate([]int{0}, "x is two", func(values []int) bool {
			return values[0] == 2
		})
		var got [][]int
		solve.Enumerate(
			[]core.Var{searchVar("x", []int{1, 2}), searchVar("y", []int{4, 5})},
			[]core.Constraint{constraint},
			func(values []int) bool {
				got = append(got, values)
				return true
			},
		)
		want := [][]int{{2, 4}, {2, 5}}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("assignments = %v, want %v", got, want)
		}
		if constraint.calls != 2 {
			t.Fatalf("Holds calls = %d, want 2; the failing prefix must not descend", constraint.calls)
		}
	})

	t.Run("visitor false stops before another constraint evaluation", func(t *testing.T) {
		constraint := searchPredicate([]int{0, 1}, "all", nil)
		visits := 0
		solve.Enumerate(
			[]core.Var{searchVar("x", []int{1, 2}), searchVar("y", []int{3, 4})},
			[]core.Constraint{constraint},
			func([]int) bool {
				visits++
				return false
			},
		)
		if visits != 1 || constraint.calls != 1 {
			t.Fatalf("visits = %d and Holds calls = %d, want 1 and 1", visits, constraint.calls)
		}
	})

	t.Run("callback assignments are independent snapshots", func(t *testing.T) {
		var retained [][]int
		var seen [][]int
		solve.Enumerate([]core.Var{searchVar("x", []int{1, 2})}, nil, func(values []int) bool {
			seen = append(seen, append([]int(nil), values...))
			retained = append(retained, values)
			if len(retained) == 1 {
				values[0] = 99
			}
			return true
		})
		if want := [][]int{{1}, {2}}; !reflect.DeepEqual(seen, want) {
			t.Fatalf("visits = %v, want %v", seen, want)
		}
		if retained[0][0] != 99 || retained[1][0] != 2 {
			t.Fatalf("callback snapshots were not independently mutable: %v", retained)
		}
	})
}

func TestFirstSolutions(t *testing.T) {
	cases := []struct {
		name   string
		model  *core.Model
		want   []int
		wantOK bool
	}{
		{
			name:   "first lexicographic assignment",
			model:  searchModel([]core.Var{searchVar("x", []int{-2, 0, 5}), searchVar("y", []int{1, 4})}, nil, nil),
			want:   []int{-2, 1},
			wantOK: true,
		},
		{
			name: "no solution",
			model: searchModel(
				[]core.Var{searchVar("x", []int{1, 2})},
				[]core.Constraint{searchPredicate([]int{0}, "false", func([]int) bool { return false })},
				nil,
			),
		},
		{
			name: "ignores unevaluable objective",
			model: searchModel(
				[]core.Var{searchVar("x", []int{1, 2})},
				nil,
				&core.Objective{E: core.Bin{Op: "/", L: core.Num{V: 1}, R: core.Num{V: 0}}},
			),
			want:   []int{1},
			wantOK: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := solve.First(tc.model)
			if ok != tc.wantOK || !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("First() = (%v, %v), want (%v, %v)", got, ok, tc.want, tc.wantOK)
			}
		})
	}

	if got, ok := solve.First(nil); ok || got != nil {
		t.Fatalf("First(nil) = (%v, %v), want (nil, false)", got, ok)
	}
}

func TestBestObjectiveCases(t *testing.T) {
	cases := []struct {
		name   string
		model  *core.Model
		want   []int
		score  int64
		wantOK bool
	}{
		{
			name:   "no objective returns first with zero score",
			model:  searchModel([]core.Var{searchVar("x", []int{2, 4})}, nil, nil),
			want:   []int{2},
			score:  0,
			wantOK: true,
		},
		{
			name:   "minimize",
			model:  searchModel([]core.Var{searchVar("x", []int{1, 2, 3})}, nil, &core.Objective{E: core.Ref{Index: 0, Name: "x"}}),
			want:   []int{1},
			score:  1,
			wantOK: true,
		},
		{
			name:   "maximize",
			model:  searchModel([]core.Var{searchVar("x", []int{1, 2, 3})}, nil, &core.Objective{Maximize: true, E: core.Ref{Index: 0, Name: "x"}}),
			want:   []int{3},
			score:  3,
			wantOK: true,
		},
		{
			name:   "minimize tie keeps first",
			model:  searchModel([]core.Var{searchVar("x", []int{1, 2, 3})}, nil, &core.Objective{E: core.Num{V: 7}}),
			want:   []int{1},
			score:  7,
			wantOK: true,
		},
		{
			name:   "maximize tie keeps first",
			model:  searchModel([]core.Var{searchVar("x", []int{1, 2, 3})}, nil, &core.Objective{Maximize: true, E: core.Num{V: 7}}),
			want:   []int{1},
			score:  7,
			wantOK: true,
		},
		{
			name:   "minimize negative and zero scores",
			model:  searchModel([]core.Var{searchVar("x", []int{-2, 0, 3})}, nil, &core.Objective{E: core.Ref{Index: 0, Name: "x"}}),
			want:   []int{-2},
			score:  -2,
			wantOK: true,
		},
		{
			name:   "maximize zero and negative scores",
			model:  searchModel([]core.Var{searchVar("x", []int{-2, 0})}, nil, &core.Objective{Maximize: true, E: core.Ref{Index: 0, Name: "x"}}),
			want:   []int{0},
			score:  0,
			wantOK: true,
		},
		{
			name: "division by zero candidate is skipped",
			model: searchModel(
				[]core.Var{searchVar("x", []int{0, 1, 2})},
				nil,
				&core.Objective{E: core.Bin{Op: "/", L: core.Num{V: 10}, R: core.Ref{Index: 0, Name: "x"}}},
			),
			want:   []int{2},
			score:  5,
			wantOK: true,
		},
		{
			name: "remainder by zero candidate is skipped",
			model: searchModel(
				[]core.Var{searchVar("x", []int{0, 3, 4})},
				nil,
				&core.Objective{E: core.Bin{Op: "%", L: core.Num{V: 10}, R: core.Ref{Index: 0, Name: "x"}}},
			),
			want:   []int{3},
			score:  1,
			wantOK: true,
		},
		{
			name: "all division objectives are unevaluable",
			model: searchModel(
				[]core.Var{searchVar("x", []int{1, 2})},
				nil,
				&core.Objective{E: core.Bin{Op: "/", L: core.Num{V: 10}, R: core.Num{V: 0}}},
			),
		},
		{
			name: "all remainder objectives are unevaluable",
			model: searchModel(
				[]core.Var{searchVar("x", []int{1, 2})},
				nil,
				&core.Objective{E: core.Bin{Op: "%", L: core.Num{V: 10}, R: core.Num{V: 0}}},
			),
		},
		{
			name: "no feasible candidates",
			model: searchModel(
				[]core.Var{searchVar("x", []int{1, 2})},
				[]core.Constraint{searchPredicate([]int{0}, "false", func([]int) bool { return false })},
				&core.Objective{E: core.Ref{Index: 0, Name: "x"}},
			),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, score, ok := solve.Best(tc.model)
			if ok != tc.wantOK || score != tc.score || !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("Best() = (%v, %d, %v), want (%v, %d, %v)", got, score, ok, tc.want, tc.score, tc.wantOK)
			}
		})
	}

	values, _, ok := solve.Best(searchModel(
		[]core.Var{searchVar("x", []int{1, 2})},
		nil,
		&core.Objective{E: core.Ref{Index: 0, Name: "x"}},
	))
	if !ok {
		t.Fatal("Best() unexpectedly found no candidate")
	}
	values[0] = 99
	next, _, nextOK := solve.Best(searchModel(
		[]core.Var{searchVar("x", []int{1, 2})},
		nil,
		&core.Objective{E: core.Ref{Index: 0, Name: "x"}},
	))
	if !nextOK || !reflect.DeepEqual(next, []int{1}) {
		t.Fatalf("Best() result was not an independent snapshot: (%v, %v)", next, nextOK)
	}
}

func TestCountBoundariesAndSnapshots(t *testing.T) {
	cases := []struct {
		name  string
		model *core.Model
		limit int
		wantN int
		more  bool
		want  [][]int
	}{
		{
			name:  "zero solutions",
			model: searchModel([]core.Var{searchVar("x", []int{1, 2})}, []core.Constraint{searchPredicate([]int{0}, "false", func([]int) bool { return false })}, nil),
			limit: 3,
		},
		{
			name:  "fewer than limit",
			model: searchModel([]core.Var{searchVar("x", []int{1, 2})}, nil, nil),
			limit: 3,
			wantN: 2,
			want:  [][]int{{1}, {2}},
		},
		{
			name:  "exactly limit",
			model: searchModel([]core.Var{searchVar("x", []int{1, 2})}, nil, nil),
			limit: 2,
			wantN: 2,
			want:  [][]int{{1}, {2}},
		},
		{
			name:  "more than limit",
			model: searchModel([]core.Var{searchVar("x", []int{1, 2, 3})}, nil, nil),
			limit: 2,
			wantN: 2,
			more:  true,
			want:  [][]int{{1}, {2}},
		},
		{
			name:  "limit one probes the second solution",
			model: searchModel([]core.Var{searchVar("x", []int{1, 2, 3})}, nil, nil),
			limit: 1,
			wantN: 1,
			more:  true,
			want:  [][]int{{1}},
		},
		{
			name:  "limit one million is accepted",
			model: searchModel([]core.Var{searchVar("x", []int{1, 2})}, nil, nil),
			limit: 1000000,
			wantN: 2,
			want:  [][]int{{1}, {2}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got [][]int
			n, more := solve.Count(tc.model, tc.limit, func(values []int) {
				got = append(got, values)
			})
			if n != tc.wantN || more != tc.more || !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("Count() = (%d, %v), visits %v; want (%d, %v), visits %v", n, more, got, tc.wantN, tc.more, tc.want)
			}
		})
	}

	t.Run("probe is not visited and search stops at solution limit plus one", func(t *testing.T) {
		constraint := searchPredicate([]int{0}, "count visits", nil)
		visits := 0
		n, more := solve.Count(
			searchModel([]core.Var{searchVar("x", []int{1, 2, 3, 4})}, []core.Constraint{constraint}, nil),
			1,
			func([]int) { visits++ },
		)
		if n != 1 || !more || visits != 1 || constraint.calls != 2 {
			t.Fatalf("Count() = (%d, %v), visits %d, Holds calls %d; want (1, true), 1, 2", n, more, visits, constraint.calls)
		}
	})

	t.Run("nil visitor counts without callbacks", func(t *testing.T) {
		n, more := solve.Count(searchModel([]core.Var{searchVar("x", []int{1, 2, 3})}, nil, nil), 3, nil)
		if n != 3 || more {
			t.Fatalf("Count(nil visitor) = (%d, %v), want (3, false)", n, more)
		}
	})

	t.Run("unevaluable objective does not affect count", func(t *testing.T) {
		model := searchModel(
			[]core.Var{searchVar("x", []int{1, 2})},
			nil,
			&core.Objective{E: core.Bin{Op: "/", L: core.Num{V: 3}, R: core.Num{V: 0}}},
		)
		if n, more := solve.Count(model, 2, nil); n != 2 || more {
			t.Fatalf("Count() with unevaluable objective = (%d, %v), want (2, false)", n, more)
		}
	})

	t.Run("visits are independent snapshots", func(t *testing.T) {
		var seen [][]int
		n, more := solve.Count(searchModel([]core.Var{searchVar("x", []int{1, 2})}, nil, nil), 2, func(values []int) {
			seen = append(seen, values)
			if len(seen) == 1 {
				values[0] = 99
			}
		})
		if n != 2 || more || seen[0][0] != 99 || seen[1][0] != 2 {
			t.Fatalf("Count() snapshot visits = (%d, %v), %v", n, more, seen)
		}
	})
}

func TestWorkedSolverExamples(t *testing.T) {
	t.Run("magic square first, second and count", func(t *testing.T) {
		model := searchModel(
			[]core.Var{
				searchVar("a", []int{1, 2, 3, 4, 5, 6, 7, 8, 9}),
				searchVar("b", []int{1, 2, 3, 4, 5, 6, 7, 8, 9}),
				searchVar("c", []int{1, 2, 3, 4, 5, 6, 7, 8, 9}),
				searchVar("d", []int{1, 2, 3, 4, 5, 6, 7, 8, 9}),
				searchVar("e", []int{1, 2, 3, 4, 5, 6, 7, 8, 9}),
				searchVar("f", []int{1, 2, 3, 4, 5, 6, 7, 8, 9}),
				searchVar("g", []int{1, 2, 3, 4, 5, 6, 7, 8, 9}),
				searchVar("h", []int{1, 2, 3, 4, 5, 6, 7, 8, 9}),
				searchVar("i", []int{1, 2, 3, 4, 5, 6, 7, 8, 9}),
			},
			[]core.Constraint{
				searchAllDifferent(0, 1, 2, 3, 4, 5, 6, 7, 8),
				searchSum(15, 0, 1, 2),
				searchSum(15, 3, 4, 5),
				searchSum(15, 6, 7, 8),
				searchSum(15, 0, 3, 6),
				searchSum(15, 1, 4, 7),
				searchSum(15, 2, 5, 8),
				searchSum(15, 0, 4, 8),
				searchSum(15, 2, 4, 6),
			},
			nil,
		)
		first, ok := solve.First(model)
		if !ok || !reflect.DeepEqual(first, []int{2, 7, 6, 9, 5, 1, 4, 3, 8}) {
			t.Fatalf("First(magic) = (%v, %v)", first, ok)
		}
		var firstTwo [][]int
		n, more := solve.Count(model, 2, func(values []int) { firstTwo = append(firstTwo, values) })
		want := [][]int{{2, 7, 6, 9, 5, 1, 4, 3, 8}, {2, 9, 4, 7, 5, 3, 6, 1, 8}}
		if n != 2 || !more || !reflect.DeepEqual(firstTwo, want) {
			t.Fatalf("Count(magic, 2) = (%d, %v), %v; want (2, true), %v", n, more, firstTwo, want)
		}
		if n, more := solve.Count(model, 100, nil); n != 8 || more {
			t.Fatalf("Count(magic, 100) = (%d, %v), want (8, false)", n, more)
		}
	})

	t.Run("shifts count and best solution", func(t *testing.T) {
		model := searchModel(
			[]core.Var{
				searchVar("ann", []int{1, 2, 3}),
				searchVar("bob", []int{1, 2, 3}),
				searchVar("cy", []int{1, 2, 3}),
				searchVar("cost", []int{10, 20, 30}),
			},
			[]core.Constraint{
				searchAllDifferent(0, 1, 2),
				searchPredicate([]int{0}, "ann != 3", func(values []int) bool { return values[0] != 3 }),
				searchPredicate([]int{2, 3}, "shift table", func(values []int) bool {
					return values[2] == 1 && values[3] == 30 ||
						values[2] == 2 && values[3] == 20 ||
						values[2] == 3 && values[3] == 10
				}),
				searchPredicate([]int{0, 1, 2}, "one early shift", func(values []int) bool {
					return (values[0] == 1) != (values[1] == 1) != (values[2] == 1)
				}),
			},
			&core.Objective{E: core.Bin{
				Op: "+",
				L:  core.Ref{Index: 3, Name: "cost"},
				R:  core.Ref{Index: 1, Name: "bob"},
			}},
		)
		if n, more := solve.Count(model, 20, nil); n != 4 || more {
			t.Fatalf("Count(shifts, 20) = (%d, %v), want (4, false)", n, more)
		}
		values, score, ok := solve.Best(model)
		if !ok || score != 11 || !reflect.DeepEqual(values, []int{2, 1, 3, 10}) {
			t.Fatalf("Best(shifts) = (%v, %d, %v), want ([2 1 3 10], 11, true)", values, score, ok)
		}
	})
}
