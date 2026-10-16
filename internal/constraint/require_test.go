package constraint

import (
	"reflect"
	"testing"

	"github.com/Gulliby-development/narrowfit/internal/core"
)

func requireTestVars() []core.Var {
	return []core.Var{
		{Name: "a", Values: []int{-7, -3, -1, 0, 1, 2, 3, 7}},
		{Name: "b", Values: []int{-7, -3, -1, 0, 1, 2, 3, 7}},
		{Name: "c", Values: []int{-7, -3, -1, 0, 1, 2, 3, 7}},
	}
}

func TestRequireBuildCases(t *testing.T) {
	tests := []struct {
		name      string
		expr      core.Expr
		values    []int
		wantText  string
		wantScope []int
		wantHolds bool
	}{
		{
			name:      "true constant has empty scope",
			expr:      core.Bin{Op: "==", L: core.Num{V: 1}, R: core.Num{V: 1}},
			values:    []int{0, 0, 0},
			wantText:  "require 1 == 1",
			wantScope: nil,
			wantHolds: true,
		},
		{
			name:      "false constant does not hold",
			expr:      core.Bin{Op: "==", L: core.Num{V: 1}, R: core.Num{V: 2}},
			values:    []int{0, 0, 0},
			wantText:  "require 1 == 2",
			wantScope: nil,
			wantHolds: false,
		},
		{
			name:      "equality comparison",
			expr:      core.Bin{Op: "==", L: core.Ref{Index: 0, Name: "a"}, R: core.Num{V: 2}},
			values:    []int{2, 0, 0},
			wantText:  "require a == 2",
			wantScope: []int{0},
			wantHolds: true,
		},
		{
			name:      "inequality comparison",
			expr:      core.Bin{Op: "!=", L: core.Ref{Index: 0, Name: "a"}, R: core.Num{V: 2}},
			values:    []int{2, 0, 0},
			wantText:  "require a != 2",
			wantScope: []int{0},
			wantHolds: false,
		},
		{
			name:      "less than comparison",
			expr:      core.Bin{Op: "<", L: core.Ref{Index: 0, Name: "a"}, R: core.Num{V: 2}},
			values:    []int{2, 0, 0},
			wantText:  "require a < 2",
			wantScope: []int{0},
			wantHolds: false,
		},
		{
			name:      "less than or equal comparison",
			expr:      core.Bin{Op: "<=", L: core.Ref{Index: 0, Name: "a"}, R: core.Num{V: 2}},
			values:    []int{2, 0, 0},
			wantText:  "require a <= 2",
			wantScope: []int{0},
			wantHolds: true,
		},
		{
			name:      "greater than comparison",
			expr:      core.Bin{Op: ">", L: core.Ref{Index: 0, Name: "a"}, R: core.Num{V: 2}},
			values:    []int{2, 0, 0},
			wantText:  "require a > 2",
			wantScope: []int{0},
			wantHolds: false,
		},
		{
			name:      "greater than or equal comparison",
			expr:      core.Bin{Op: ">=", L: core.Ref{Index: 0, Name: "a"}, R: core.Num{V: 2}},
			values:    []int{2, 0, 0},
			wantText:  "require a >= 2",
			wantScope: []int{0},
			wantHolds: true,
		},
		{
			name: "repeated references are deduplicated in scope",
			expr: core.Bin{
				Op: "and",
				L:  core.Bin{Op: "==", L: core.Ref{Index: 0, Name: "a"}, R: core.Ref{Index: 0, Name: "a"}},
				R:  core.Bin{Op: ">=", L: core.Ref{Index: 0, Name: "a"}, R: core.Ref{Index: 0, Name: "a"}},
			},
			values:    []int{2, 0, 0},
			wantText:  "require a == a and a >= a",
			wantScope: []int{0},
			wantHolds: true,
		},
		{
			name: "scope is sorted and unique across logical operands",
			expr: core.Bin{
				Op: "and",
				L:  core.Bin{Op: ">", L: core.Ref{Index: 2, Name: "c"}, R: core.Ref{Index: 0, Name: "a"}},
				R: core.Bin{
					Op: "and",
					L:  core.Bin{Op: "!=", L: core.Ref{Index: 1, Name: "b"}, R: core.Ref{Index: 2, Name: "c"}},
					R:  core.Bin{Op: "<", L: core.Ref{Index: 0, Name: "a"}, R: core.Ref{Index: 1, Name: "b"}},
				},
			},
			values:    []int{1, 2, 3},
			wantText:  "require c > a and (b != c and a < b)",
			wantScope: []int{0, 1, 2},
			wantHolds: true,
		},
		{
			name: "and requires both conditions",
			expr: core.Bin{
				Op: "and",
				L:  core.Bin{Op: ">", L: core.Ref{Index: 0, Name: "a"}, R: core.Num{V: 0}},
				R:  core.Bin{Op: ">", L: core.Ref{Index: 1, Name: "b"}, R: core.Num{V: 0}},
			},
			values:    []int{1, 2, 0},
			wantText:  "require a > 0 and b > 0",
			wantScope: []int{0, 1},
			wantHolds: true,
		},
		{
			name: "or accepts either condition",
			expr: core.Bin{
				Op: "or",
				L:  core.Bin{Op: ">", L: core.Ref{Index: 0, Name: "a"}, R: core.Num{V: 0}},
				R:  core.Bin{Op: ">", L: core.Ref{Index: 1, Name: "b"}, R: core.Num{V: 0}},
			},
			values:    []int{-1, 2, 0},
			wantText:  "require a > 0 or b > 0",
			wantScope: []int{0, 1},
			wantHolds: true,
		},
		{
			name: "not negates a false condition",
			expr: core.Not{X: core.Bin{
				Op: "==", L: core.Ref{Index: 0, Name: "a"}, R: core.Num{V: 2},
			}},
			values:    []int{1, 0, 0},
			wantText:  "require not a == 2",
			wantScope: []int{0},
			wantHolds: true,
		},
		{
			name: "not parenthesizes a weaker conjunction",
			expr: core.Not{X: core.Bin{
				Op: "and",
				L:  core.Bin{Op: ">", L: core.Ref{Index: 0, Name: "a"}, R: core.Ref{Index: 1, Name: "b"}},
				R:  core.Bin{Op: "<", L: core.Ref{Index: 1, Name: "b"}, R: core.Ref{Index: 2, Name: "c"}},
			}},
			values:    []int{3, 2, 3},
			wantText:  "require not (a > b and b < c)",
			wantScope: []int{0, 1, 2},
			wantHolds: false,
		},
		{
			name: "implication left child is parenthesized",
			expr: core.Bin{
				Op: "->",
				L: core.Bin{
					Op: "->",
					L:  core.Bin{Op: ">", L: core.Ref{Index: 0, Name: "a"}, R: core.Ref{Index: 1, Name: "b"}},
					R:  core.Bin{Op: ">", L: core.Ref{Index: 1, Name: "b"}, R: core.Ref{Index: 2, Name: "c"}},
				},
				R: core.Bin{Op: ">", L: core.Ref{Index: 0, Name: "a"}, R: core.Ref{Index: 2, Name: "c"}},
			},
			values:    []int{3, 2, 1},
			wantText:  "require (a > b -> b > c) -> a > c",
			wantScope: []int{0, 1, 2},
			wantHolds: true,
		},
		{
			name: "implication right child remains right associative",
			expr: core.Bin{
				Op: "->",
				L:  core.Bin{Op: ">", L: core.Ref{Index: 0, Name: "a"}, R: core.Ref{Index: 1, Name: "b"}},
				R: core.Bin{
					Op: "->",
					L:  core.Bin{Op: ">", L: core.Ref{Index: 1, Name: "b"}, R: core.Ref{Index: 2, Name: "c"}},
					R:  core.Bin{Op: ">", L: core.Ref{Index: 0, Name: "a"}, R: core.Ref{Index: 2, Name: "c"}},
				},
			},
			values:    []int{3, 2, 1},
			wantText:  "require a > b -> b > c -> a > c",
			wantScope: []int{0, 1, 2},
			wantHolds: true,
		},
		{
			name: "arithmetic comparison evaluates to true",
			expr: core.Bin{
				Op: "==",
				L: core.Bin{
					Op: "-",
					L:  core.Ref{Index: 0, Name: "a"},
					R:  core.Ref{Index: 1, Name: "b"},
				},
				R: core.Num{V: -1},
			},
			values:    []int{1, 2, 0},
			wantText:  "require a - b == -1",
			wantScope: []int{0, 1},
			wantHolds: true,
		},
		{
			name: "evaluated division by zero violates the requirement",
			expr: core.Bin{
				Op: "==",
				L:  core.Bin{Op: "/", L: core.Ref{Index: 0, Name: "a"}, R: core.Num{V: 0}},
				R:  core.Num{V: 1},
			},
			values:    []int{1, 0, 0},
			wantText:  "require a / 0 == 1",
			wantScope: []int{0},
			wantHolds: false,
		},
		{
			name: "evaluated remainder by zero violates the requirement",
			expr: core.Bin{
				Op: "==",
				L:  core.Bin{Op: "%", L: core.Ref{Index: 0, Name: "a"}, R: core.Num{V: 0}},
				R:  core.Num{V: 1},
			},
			values:    []int{1, 0, 0},
			wantText:  "require a % 0 == 1",
			wantScope: []int{0},
			wantHolds: false,
		},
		{
			name: "not cannot turn unevaluability into a true condition",
			expr: core.Not{X: core.Bin{
				Op: "==",
				L:  core.Bin{Op: "/", L: core.Ref{Index: 0, Name: "a"}, R: core.Num{V: 0}},
				R:  core.Num{V: 1},
			}},
			values:    []int{1, 0, 0},
			wantText:  "require not a / 0 == 1",
			wantScope: []int{0},
			wantHolds: false,
		},
		{
			name: "false and skips an unevaluable right condition",
			expr: core.Bin{
				Op: "and",
				L:  core.Bin{Op: "==", L: core.Num{V: 1}, R: core.Num{V: 2}},
				R: core.Bin{
					Op: "==",
					L:  core.Bin{Op: "/", L: core.Ref{Index: 0, Name: "a"}, R: core.Num{V: 0}},
					R:  core.Num{V: 1},
				},
			},
			values:    []int{1, 0, 0},
			wantText:  "require 1 == 2 and a / 0 == 1",
			wantScope: []int{0},
			wantHolds: false,
		},
		{
			name: "true or skips an unevaluable right condition",
			expr: core.Bin{
				Op: "or",
				L:  core.Bin{Op: "==", L: core.Num{V: 1}, R: core.Num{V: 1}},
				R: core.Bin{
					Op: "==",
					L:  core.Bin{Op: "%", L: core.Ref{Index: 0, Name: "a"}, R: core.Num{V: 0}},
					R:  core.Num{V: 1},
				},
			},
			values:    []int{1, 0, 0},
			wantText:  "require 1 == 1 or a % 0 == 1",
			wantScope: []int{0},
			wantHolds: true,
		},
		{
			name: "false implication skips an unevaluable right condition",
			expr: core.Bin{
				Op: "->",
				L:  core.Bin{Op: "==", L: core.Num{V: 1}, R: core.Num{V: 2}},
				R: core.Bin{
					Op: "==",
					L:  core.Bin{Op: "/", L: core.Ref{Index: 0, Name: "a"}, R: core.Num{V: 0}},
					R:  core.Num{V: 1},
				},
			},
			values:    []int{1, 0, 0},
			wantText:  "require 1 == 2 -> a / 0 == 1",
			wantScope: []int{0},
			wantHolds: true,
		},
		{
			name: "positive division is evaluated",
			expr: core.Bin{
				Op: "==",
				L:  core.Bin{Op: "/", L: core.Ref{Index: 0, Name: "a"}, R: core.Num{V: 2}},
				R:  core.Num{V: 1},
			},
			values:    []int{3, 0, 0},
			wantText:  "require a / 2 == 1",
			wantScope: []int{0},
			wantHolds: true,
		},
	}

	vars := requireTestVars()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := NewRequire(test.expr)
			if got == nil {
				t.Fatal("NewRequire(...) = nil, want a constraint")
			}
			if text := got.Text(); text != test.wantText {
				t.Errorf("Text() = %q, want %q", text, test.wantText)
			}
			if scope := got.Scope(); !reflect.DeepEqual(scope, test.wantScope) {
				t.Errorf("Scope() = %v, want %v", scope, test.wantScope)
			}
			if holds := got.Holds(test.values); holds != test.wantHolds {
				t.Errorf("Holds(%v) = %v, want %v", test.values, holds, test.wantHolds)
			}
			if len(test.values) != len(vars) {
				t.Fatalf("test assignment has %d values, want complete assignment of %d", len(test.values), len(vars))
			}
		})
	}
}
