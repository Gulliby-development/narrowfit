package constraint

import (
	"reflect"
	"testing"

	"github.com/Gulliby-development/narrowfit/internal/core"
)

func tableTestVars() []core.Var {
	return []core.Var{
		{Name: "a", Values: []int{-1000, -1, 0, 1, 2, 1000}},
		{Name: "b", Values: []int{-1000, -1, 0, 1, 2, 1000}},
		{Name: "c", Values: []int{-1000, -1, 0, 1, 2, 1000}},
	}
}

func TestTableBuildCases(t *testing.T) {
	vars := tableTestVars()
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
			name:           "one variable and one tuple",
			words:          []string{"a", ":", "1"},
			wantConstraint: true,
			wantText:       "table a : 1",
			wantScope:      []int{0},
			values:         []int{1, 2, -1},
			wantHolds:      true,
		},
		{
			name:           "standalone delimiters preserve tuple order",
			words:          []string{"a", "b", ":", "1", "2", "|", "3", "4"},
			wantConstraint: true,
			wantText:       "table a b : 1 2 | 3 4",
			wantScope:      []int{0, 1},
			values:         []int{3, 4, -1},
			wantHolds:      true,
		},
		{
			name:           "listed columns are independent of sorted scope",
			words:          []string{"b", "a", ":", "1", "2"},
			wantConstraint: true,
			wantText:       "table b a : 1 2",
			wantScope:      []int{0, 1},
			values:         []int{2, 1, 0},
			wantHolds:      true,
		},
		{
			name:           "listed column order is used when assignment differs",
			words:          []string{"b", "a", ":", "1", "2"},
			wantConstraint: true,
			wantText:       "table b a : 1 2",
			wantScope:      []int{0, 1},
			values:         []int{1, 2, 0},
			wantHolds:      false,
		},
		{
			name:           "minimum integer boundary is accepted",
			words:          []string{"a", ":", "-1000"},
			wantConstraint: true,
			wantText:       "table a : -1000",
			wantScope:      []int{0},
			values:         []int{-1000, 1, 0},
			wantHolds:      true,
		},
		{
			name:           "maximum integer boundary is accepted",
			words:          []string{"a", ":", "1000"},
			wantConstraint: true,
			wantText:       "table a : 1000",
			wantScope:      []int{0},
			values:         []int{1000, -1, 0},
			wantHolds:      true,
		},
		{
			name:           "assignment may match a later tuple",
			words:          []string{"a", "b", ":", "1", "2", "|", "3", "4"},
			wantConstraint: true,
			wantText:       "table a b : 1 2 | 3 4",
			wantScope:      []int{0, 1},
			values:         []int{1, 2, 1000},
			wantHolds:      true,
		},
		{
			name:           "assignment outside every tuple does not hold",
			words:          []string{"a", "b", ":", "1", "2", "|", "3", "4"},
			wantConstraint: true,
			wantText:       "table a b : 1 2 | 3 4",
			wantScope:      []int{0, 1},
			values:         []int{1, 4, 0},
			wantHolds:      false,
		},
		{
			name:           "variables outside scope do not affect matching",
			words:          []string{"a", "b", ":", "1", "2"},
			wantConstraint: true,
			wantText:       "table a b : 1 2",
			wantScope:      []int{0, 1},
			values:         []int{1, 2, 1000},
			wantHolds:      true,
		},
		{
			name:        "no words is invalid",
			words:       []string{},
			wantMessage: "invalid table statement",
		},
		{
			name:        "standalone colon without a name is invalid",
			words:       []string{":", "1"},
			wantMessage: "invalid table statement",
		},
		{
			name:        "standalone bar without table syntax is invalid",
			words:       []string{"|", ":", "1"},
			wantMessage: "invalid table statement",
		},
		{
			name:        "missing colon is invalid",
			words:       []string{"a", "1"},
			wantMessage: "invalid table statement",
		},
		{
			name:        "missing tuple after colon is invalid",
			words:       []string{"a", ":"},
			wantMessage: "invalid table statement",
		},
		{
			name:        "empty first tuple is invalid",
			words:       []string{"a", ":", "|", "1"},
			wantMessage: "invalid table statement",
		},
		{
			name:        "empty last tuple is invalid",
			words:       []string{"a", ":", "1", "|"},
			wantMessage: "invalid table statement",
		},
		{
			name:        "empty tuple between separators is invalid",
			words:       []string{"a", ":", "1", "|", "|", "2"},
			wantMessage: "invalid table statement",
		},
		{
			name:        "tuple narrower than listed names is invalid",
			words:       []string{"a", "b", ":", "1"},
			wantMessage: "invalid table statement",
		},
		{
			name:        "tuple wider than listed names is invalid",
			words:       []string{"a", ":", "1", "2"},
			wantMessage: "invalid table statement",
		},
		{
			name:        "minimum integer below the boundary fails before name lookup",
			words:       []string{"ghost", ":", "-1001"},
			wantMessage: "invalid table statement",
		},
		{
			name:        "maximum integer above the boundary fails before name lookup",
			words:       []string{"ghost", ":", "1001"},
			wantMessage: "invalid table statement",
		},
		{
			name:        "plus sign is not a valid table integer",
			words:       []string{"ghost", ":", "+1"},
			wantMessage: "invalid table statement",
		},
		{
			name:        "negative zero is not a valid table integer",
			words:       []string{"ghost", ":", "-0"},
			wantMessage: "invalid table statement",
		},
		{
			name:        "leading zero is not a valid table integer",
			words:       []string{"ghost", ":", "01"},
			wantMessage: "invalid table statement",
		},
		{
			name:        "range is not a table integer",
			words:       []string{"ghost", ":", "1..2"},
			wantMessage: "invalid table statement",
		},
		{
			name:        "duplicate colon is invalid",
			words:       []string{"a", ":", "1", ":", "2"},
			wantMessage: "invalid table statement",
		},
		{
			name:        "malformed tuple width precedes unknown name",
			words:       []string{"ghost", ":", "1", "2"},
			wantMessage: "invalid table statement",
		},
		{
			name:        "first unknown name precedes a later repeat",
			words:       []string{"a", "ghost", "a", ":", "1", "2", "3"},
			wantMessage: "unknown variable \"ghost\"",
		},
		{
			name:        "first unknown name is reported",
			words:       []string{"ghost", "missing", ":", "1", "2"},
			wantMessage: "unknown variable \"ghost\"",
		},
		{
			name:        "first repeated name is reported",
			words:       []string{"a", "b", "a", ":", "1", "2", "3"},
			wantMessage: "repeated variable \"a\"",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, msg, known := Build("table", test.words, vars)
			if !known {
				t.Fatal("Build(...).known = false, want true for table")
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
