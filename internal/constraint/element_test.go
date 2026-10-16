package constraint

import (
	"reflect"
	"testing"

	"github.com/Gulliby-development/narrowfit/internal/core"
)

func elementTestVars() []core.Var {
	return []core.Var{
		{Name: "i", Values: []int{-1, 0, 1, 2, 3, 4, 10, 20, 30, 99}},
		{Name: "a", Values: []int{-1, 0, 1, 2, 3, 4, 10, 20, 30, 99}},
		{Name: "b", Values: []int{-1, 0, 1, 2, 3, 4, 10, 20, 30, 99}},
		{Name: "c", Values: []int{-1, 0, 1, 2, 3, 4, 10, 20, 30, 99}},
		{Name: "r", Values: []int{-1, 0, 1, 2, 3, 4, 10, 20, 30, 99}},
		{Name: "z", Values: []int{-1, 0, 1, 2, 3, 4, 10, 20, 30, 99}},
	}
}

func TestElementBuildCases(t *testing.T) {
	vars := elementTestVars()
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
			name:           "singleton list selects its only value",
			words:          []string{"i", "in", "a", "is", "r"},
			wantConstraint: true,
			wantText:       "element i in a is r",
			wantScope:      []int{0, 1, 4},
			values:         []int{1, 10, 20, 30, 10, 99},
			wantHolds:      true,
		},
		{
			name:           "singleton list rejects a different result",
			words:          []string{"i", "in", "a", "is", "r"},
			wantConstraint: true,
			wantText:       "element i in a is r",
			wantScope:      []int{0, 1, 4},
			values:         []int{1, 10, 20, 30, 20, 99},
			wantHolds:      false,
		},
		{
			name:           "index one selects the first listed variable",
			words:          []string{"i", "in", "a", "b", "c", "is", "r"},
			wantConstraint: true,
			wantText:       "element i in a b c is r",
			wantScope:      []int{0, 1, 2, 3, 4},
			values:         []int{1, 10, 20, 30, 10, 99},
			wantHolds:      true,
		},
		{
			name:           "middle index selects the middle listed variable",
			words:          []string{"i", "in", "a", "b", "c", "is", "r"},
			wantConstraint: true,
			wantText:       "element i in a b c is r",
			wantScope:      []int{0, 1, 2, 3, 4},
			values:         []int{2, 10, 20, 30, 20, 99},
			wantHolds:      true,
		},
		{
			name:           "last index selects the last listed variable",
			words:          []string{"i", "in", "a", "b", "c", "is", "r"},
			wantConstraint: true,
			wantText:       "element i in a b c is r",
			wantScope:      []int{0, 1, 2, 3, 4},
			values:         []int{3, 10, 20, 30, 30, 99},
			wantHolds:      true,
		},
		{
			name:           "selection follows listed order rather than scope order",
			words:          []string{"i", "in", "c", "a", "b", "is", "r"},
			wantConstraint: true,
			wantText:       "element i in c a b is r",
			wantScope:      []int{0, 1, 2, 3, 4},
			values:         []int{2, 10, 20, 30, 10, 99},
			wantHolds:      true,
		},
		{
			name:           "zero index is outside the one-based list",
			words:          []string{"i", "in", "a", "b", "c", "is", "r"},
			wantConstraint: true,
			wantText:       "element i in a b c is r",
			wantScope:      []int{0, 1, 2, 3, 4},
			values:         []int{0, 10, 20, 30, 10, 99},
			wantHolds:      false,
		},
		{
			name:           "negative index is outside the one-based list",
			words:          []string{"i", "in", "a", "b", "c", "is", "r"},
			wantConstraint: true,
			wantText:       "element i in a b c is r",
			wantScope:      []int{0, 1, 2, 3, 4},
			values:         []int{-1, 10, 20, 30, 10, 99},
			wantHolds:      false,
		},
		{
			name:           "index above list length is rejected",
			words:          []string{"i", "in", "a", "b", "c", "is", "r"},
			wantConstraint: true,
			wantText:       "element i in a b c is r",
			wantScope:      []int{0, 1, 2, 3, 4},
			values:         []int{4, 10, 20, 30, 30, 99},
			wantHolds:      false,
		},
		{
			name:           "selected value must equal the result",
			words:          []string{"i", "in", "a", "b", "c", "is", "r"},
			wantConstraint: true,
			wantText:       "element i in a b c is r",
			wantScope:      []int{0, 1, 2, 3, 4},
			values:         []int{2, 10, 20, 30, 30, 99},
			wantHolds:      false,
		},
		{
			name:           "variable outside scope does not affect selection",
			words:          []string{"i", "in", "a", "b", "c", "is", "r"},
			wantConstraint: true,
			wantText:       "element i in a b c is r",
			wantScope:      []int{0, 1, 2, 3, 4},
			values:         []int{1, 10, 20, 30, 10, -1},
			wantHolds:      true,
		},
		{
			name:           "index may also appear in the middle list",
			words:          []string{"i", "in", "i", "a", "b", "is", "r"},
			wantConstraint: true,
			wantText:       "element i in i a b is r",
			wantScope:      []int{0, 1, 2, 4},
			values:         []int{1, 10, 20, 30, 1, 99},
			wantHolds:      true,
		},
		{
			name:           "result may also appear in the middle list",
			words:          []string{"i", "in", "a", "r", "c", "is", "r"},
			wantConstraint: true,
			wantText:       "element i in a r c is r",
			wantScope:      []int{0, 1, 3, 4},
			values:         []int{2, 10, 20, 30, 20, 99},
			wantHolds:      true,
		},
		{
			name:           "index and result may be the same variable",
			words:          []string{"i", "in", "a", "b", "c", "is", "i"},
			wantConstraint: true,
			wantText:       "element i in a b c is i",
			wantScope:      []int{0, 1, 2, 3},
			values:         []int{2, 10, 2, 30, 20, 99},
			wantHolds:      true,
		},
		{
			name:           "index and result may both appear in the middle list",
			words:          []string{"i", "in", "i", "a", "b", "is", "i"},
			wantConstraint: true,
			wantText:       "element i in i a b is i",
			wantScope:      []int{0, 1, 2},
			values:         []int{1, 10, 20, 30, 20, 99},
			wantHolds:      true,
		},
		{
			name:        "empty words are invalid",
			words:       []string{},
			wantMessage: "invalid element statement",
		},
		{
			name:        "too few words are invalid",
			words:       []string{"i", "in", "a", "is"},
			wantMessage: "invalid element statement",
		},
		{
			name:        "incorrect in token is rejected before name lookup",
			words:       []string{"ghost", "not-in", "a", "is", "r"},
			wantMessage: "invalid element statement",
		},
		{
			name:        "incorrect is token is rejected",
			words:       []string{"i", "in", "a", "not-is", "r"},
			wantMessage: "invalid element statement",
		},
		{
			name:        "empty middle list is invalid",
			words:       []string{"i", "in", "is", "r"},
			wantMessage: "invalid element statement",
		},
		{
			name:        "extra trailing word is invalid",
			words:       []string{"i", "in", "a", "is", "r", "extra"},
			wantMessage: "invalid element statement",
		},
		{
			name:        "unknown index is reported first",
			words:       []string{"ghost", "in", "missing", "is", "none"},
			wantMessage: "unknown variable \"ghost\"",
		},
		{
			name:        "first unknown middle variable is reported",
			words:       []string{"i", "in", "missing", "ghost", "is", "r"},
			wantMessage: "unknown variable \"missing\"",
		},
		{
			name:        "unknown result is reported after known preceding names",
			words:       []string{"i", "in", "a", "is", "missing"},
			wantMessage: "unknown variable \"missing\"",
		},
		{
			name:        "first textual unknown beats a repeated middle name",
			words:       []string{"i", "in", "ghost", "a", "a", "is", "r"},
			wantMessage: "unknown variable \"ghost\"",
		},
		{
			name:        "unknown result beats a repeated middle name",
			words:       []string{"i", "in", "a", "a", "is", "ghost"},
			wantMessage: "unknown variable \"ghost\"",
		},
		{
			name:        "middle list cannot repeat its first variable",
			words:       []string{"i", "in", "a", "a", "is", "r"},
			wantMessage: "repeated variable \"a\"",
		},
		{
			name:        "first repeated middle name is reported",
			words:       []string{"i", "in", "a", "b", "a", "b", "is", "r"},
			wantMessage: "repeated variable \"a\"",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, msg, known := Build("element", test.words, vars)
			if !known {
				t.Fatal("Build(...).known = false, want true for element")
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
