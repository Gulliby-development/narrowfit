package constraint

import (
	"reflect"
	"testing"

	"github.com/Gulliby-development/narrowfit/internal/core"
)

func countTestVars() []core.Var {
	return []core.Var{
		{Name: "a", Values: []int{-1000, -1, 0, 1, 2, 1000}},
		{Name: "b", Values: []int{-1000, -1, 0, 1, 2, 1000}},
		{Name: "c", Values: []int{-1000, -1, 0, 1, 2, 1000}},
		{Name: "d", Values: []int{-1000, -1, 0, 1, 2, 1000}},
	}
}

func TestCountBuildCases(t *testing.T) {
	vars := countTestVars()
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
			name:           "equality counts matches",
			words:          []string{"2", "a", "b", "c", "==", "2"},
			wantConstraint: true,
			wantText:       "count 2 a b c == 2",
			wantScope:      []int{0, 1, 2},
			values:         []int{2, 2, 1, 1000},
			wantHolds:      true,
		},
		{
			name:           "not equal holds when there are matches",
			words:          []string{"2", "a", "b", "c", "!=", "0"},
			wantConstraint: true,
			wantText:       "count 2 a b c != 0",
			wantScope:      []int{0, 1, 2},
			values:         []int{2, 1, 1, 1000},
			wantHolds:      true,
		},
		{
			name:           "less than is false at equality",
			words:          []string{"2", "a", "b", "c", "<", "2"},
			wantConstraint: true,
			wantText:       "count 2 a b c < 2",
			wantScope:      []int{0, 1, 2},
			values:         []int{2, 2, 1, 1000},
			wantHolds:      false,
		},
		{
			name:           "less than or equal includes equality",
			words:          []string{"2", "a", "b", "c", "<=", "2"},
			wantConstraint: true,
			wantText:       "count 2 a b c <= 2",
			wantScope:      []int{0, 1, 2},
			values:         []int{2, 2, 1, 1000},
			wantHolds:      true,
		},
		{
			name:           "greater than is false at equality",
			words:          []string{"2", "a", "b", "c", ">", "2"},
			wantConstraint: true,
			wantText:       "count 2 a b c > 2",
			wantScope:      []int{0, 1, 2},
			values:         []int{2, 2, 1, 1000},
			wantHolds:      false,
		},
		{
			name:           "greater than holds below the match count",
			words:          []string{"2", "a", "b", "c", ">", "1"},
			wantConstraint: true,
			wantText:       "count 2 a b c > 1",
			wantScope:      []int{0, 1, 2},
			values:         []int{2, 2, 1, 1000},
			wantHolds:      true,
		},
		{
			name:           "greater than or equal includes equality",
			words:          []string{"2", "a", "b", "c", ">=", "2"},
			wantConstraint: true,
			wantText:       "count 2 a b c >= 2",
			wantScope:      []int{0, 1, 2},
			values:         []int{2, 2, 1, 1000},
			wantHolds:      true,
		},
		{
			name:           "text preserves listed variable order while scope is sorted",
			words:          []string{"2", "c", "a", "b", "==", "2"},
			wantConstraint: true,
			wantText:       "count 2 c a b == 2",
			wantScope:      []int{0, 1, 2},
			values:         []int{2, 2, 1, 1000},
			wantHolds:      true,
		},
		{
			name:           "minimum match value is inclusive",
			words:          []string{"-1000", "a", "==", "1"},
			wantConstraint: true,
			wantText:       "count -1000 a == 1",
			wantScope:      []int{0},
			values:         []int{-1000, 2, 1, 1000},
			wantHolds:      true,
		},
		{
			name:           "maximum match value and target count are inclusive",
			words:          []string{"1000", "a", "<=", "1000"},
			wantConstraint: true,
			wantText:       "count 1000 a <= 1000",
			wantScope:      []int{0},
			values:         []int{1000, 2, 1, -1000},
			wantHolds:      true,
		},
		{
			name:           "zero is a valid target count",
			words:          []string{"5", "a", "==", "0"},
			wantConstraint: true,
			wantText:       "count 5 a == 0",
			wantScope:      []int{0},
			values:         []int{2, 2, 1, 1000},
			wantHolds:      true,
		},
		{
			name:           "no matching values violates a positive requested count",
			words:          []string{"5", "a", "b", "c", "==", "1"},
			wantConstraint: true,
			wantText:       "count 5 a b c == 1",
			wantScope:      []int{0, 1, 2},
			values:         []int{2, 1, 1, 1000},
			wantHolds:      false,
		},
		{
			name:        "no words is an invalid statement",
			words:       []string{},
			wantMessage: "invalid count statement",
		},
		{
			name:        "missing variable name is an invalid statement",
			words:       []string{"2", "==", "1"},
			wantMessage: "invalid count statement",
		},
		{
			name:        "invalid low target is rejected before name lookup",
			words:       []string{"-1001", "ghost", "==", "1"},
			wantMessage: "invalid count statement",
		},
		{
			name:        "invalid high target is rejected before name lookup",
			words:       []string{"1001", "ghost", "==", "1"},
			wantMessage: "invalid count statement",
		},
		{
			name:        "negative target count is invalid before name lookup",
			words:       []string{"2", "ghost", "==", "-1"},
			wantMessage: "invalid count statement",
		},
		{
			name:        "target count above the limit is invalid before name lookup",
			words:       []string{"2", "ghost", "==", "1001"},
			wantMessage: "invalid count statement",
		},
		{
			name:        "invalid operator is rejected before name lookup",
			words:       []string{"2", "ghost", "equals", "1"},
			wantMessage: "invalid count statement",
		},
		{
			name:        "leading zero is not an integer target",
			words:       []string{"02", "ghost", "==", "1"},
			wantMessage: "invalid count statement",
		},
		{
			name:        "plus sign is not an integer target",
			words:       []string{"+2", "ghost", "==", "1"},
			wantMessage: "invalid count statement",
		},
		{
			name:        "negative zero is not an integer target",
			words:       []string{"-0", "ghost", "==", "1"},
			wantMessage: "invalid count statement",
		},
		{
			name:        "first unknown name precedes a later repeat",
			words:       []string{"2", "a", "ghost", "a", "==", "1"},
			wantMessage: "unknown variable \"ghost\"",
		},
		{
			name:        "first unknown name is reported",
			words:       []string{"2", "ghost", "missing", "==", "1"},
			wantMessage: "unknown variable \"ghost\"",
		},
		{
			name:        "first repeated name is reported",
			words:       []string{"2", "a", "b", "a", "==", "1"},
			wantMessage: "repeated variable \"a\"",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, msg, known := Build("count", test.words, vars)
			if !known {
				t.Fatal("Build(...).known = false, want true for count")
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
