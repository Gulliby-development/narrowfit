package constraint

import (
	"reflect"
	"testing"

	"github.com/Gulliby-development/narrowfit/internal/core"
)

func sumTestVars() []core.Var {
	return []core.Var{
		{Name: "a", Values: []int{-1000, 1, 1000}},
		{Name: "b", Values: []int{-1000, 2, 1000}},
		{Name: "c", Values: []int{-1000, 3, 1000}},
	}
}

func TestSumBuildCases(t *testing.T) {
	vars := sumTestVars()
	tests := []struct {
		name           string
		words          []string
		wantMessage    string
		wantConstraint bool
		wantText       string
		wantScope      []int
		values         []int
		wantHolds      bool
	}{
		{
			name:           "equality holds and preserves text name order",
			words:          []string{"c", "a", "b", "==", "6"},
			wantConstraint: true,
			wantText:       "sum c a b == 6",
			wantScope:      []int{0, 1, 2},
			values:         []int{1, 2, 3},
			wantHolds:      true,
		},
		{
			name:           "inequality holds below its target",
			words:          []string{"c", "a", "b", "!=", "7"},
			wantConstraint: true,
			wantText:       "sum c a b != 7",
			wantScope:      []int{0, 1, 2},
			values:         []int{1, 2, 3},
			wantHolds:      true,
		},
		{
			name:           "less than is false at equality",
			words:          []string{"c", "a", "b", "<", "6"},
			wantConstraint: true,
			wantText:       "sum c a b < 6",
			wantScope:      []int{0, 1, 2},
			values:         []int{1, 2, 3},
			wantHolds:      false,
		},
		{
			name:           "less than holds above the sum",
			words:          []string{"c", "a", "b", "<", "7"},
			wantConstraint: true,
			wantText:       "sum c a b < 7",
			wantScope:      []int{0, 1, 2},
			values:         []int{1, 2, 3},
			wantHolds:      true,
		},
		{
			name:           "less than or equal includes the boundary",
			words:          []string{"c", "a", "b", "<=", "6"},
			wantConstraint: true,
			wantText:       "sum c a b <= 6",
			wantScope:      []int{0, 1, 2},
			values:         []int{1, 2, 3},
			wantHolds:      true,
		},
		{
			name:           "greater than is false at equality",
			words:          []string{"c", "a", "b", ">", "6"},
			wantConstraint: true,
			wantText:       "sum c a b > 6",
			wantScope:      []int{0, 1, 2},
			values:         []int{1, 2, 3},
			wantHolds:      false,
		},
		{
			name:           "greater than holds below the sum",
			words:          []string{"c", "a", "b", ">", "5"},
			wantConstraint: true,
			wantText:       "sum c a b > 5",
			wantScope:      []int{0, 1, 2},
			values:         []int{1, 2, 3},
			wantHolds:      true,
		},
		{
			name:           "greater than or equal includes the boundary",
			words:          []string{"c", "a", "b", ">=", "6"},
			wantConstraint: true,
			wantText:       "sum c a b >= 6",
			wantScope:      []int{0, 1, 2},
			values:         []int{1, 2, 3},
			wantHolds:      true,
		},
		{
			name:           "equality rejects a different sum",
			words:          []string{"a", "b", "c", "==", "5"},
			wantConstraint: true,
			wantText:       "sum a b c == 5",
			wantScope:      []int{0, 1, 2},
			values:         []int{1, 2, 3},
			wantHolds:      false,
		},
		{
			name:           "one operand is valid",
			words:          []string{"a", "==", "1"},
			wantConstraint: true,
			wantText:       "sum a == 1",
			wantScope:      []int{0},
			values:         []int{1, 2, 3},
			wantHolds:      true,
		},
		{
			name:           "inclusive minimum target is valid",
			words:          []string{"a", ">=", "-1000000"},
			wantConstraint: true,
			wantText:       "sum a >= -1000000",
			wantScope:      []int{0},
			values:         []int{-1000, 2, 3},
			wantHolds:      true,
		},
		{
			name:           "inclusive maximum target is valid",
			words:          []string{"a", "<=", "1000000"},
			wantConstraint: true,
			wantText:       "sum a <= 1000000",
			wantScope:      []int{0},
			values:         []int{1000, 2, 3},
			wantHolds:      true,
		},
		{
			name:        "missing operand is invalid before name resolution",
			words:       []string{"==", "1"},
			wantMessage: "invalid sum statement",
		},
		{
			name:        "invalid operator is rejected before unknown names",
			words:       []string{"ghost", "equals", "1"},
			wantMessage: "invalid sum statement",
		},
		{
			name:        "minimum target below range is invalid before names",
			words:       []string{"ghost", "==", "-1000001"},
			wantMessage: "invalid sum statement",
		},
		{
			name:        "maximum target above range is invalid before names",
			words:       []string{"ghost", "==", "1000001"},
			wantMessage: "invalid sum statement",
		},
		{
			name:        "plus sign is not an integer",
			words:       []string{"ghost", "==", "+3"},
			wantMessage: "invalid sum statement",
		},
		{
			name:        "leading zero is not an integer",
			words:       []string{"ghost", "==", "007"},
			wantMessage: "invalid sum statement",
		},
		{
			name:        "negative zero is not an integer",
			words:       []string{"ghost", "==", "-0"},
			wantMessage: "invalid sum statement",
		},
		{
			name:        "first unknown name precedes a later repeated name",
			words:       []string{"a", "ghost", "a", "==", "1"},
			wantMessage: "unknown variable \"ghost\"",
		},
		{
			name:        "first repeated name is reported",
			words:       []string{"a", "b", "a", "==", "1"},
			wantMessage: "repeated variable \"a\"",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, msg, known := Build("sum", test.words, vars)
			if !known {
				t.Fatal("Build(...).known = false, want true for sum")
			}
			if msg != test.wantMessage {
				t.Fatalf("Build(...).message = %q, want %q", msg, test.wantMessage)
			}
			if !test.wantConstraint {
				if got != nil {
					t.Fatalf("Build(...) constraint = %#v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatal("Build(...) constraint = nil, want a constraint")
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
		})
	}
}
