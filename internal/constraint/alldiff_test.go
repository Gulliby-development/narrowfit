package constraint

import (
	"reflect"
	"testing"

	"github.com/Gulliby-development/narrowfit/internal/core"
)

func alldiffTestVars() []core.Var {
	return []core.Var{
		{Name: "a", Values: []int{1, 2, 3}},
		{Name: "b", Values: []int{1, 2, 3}},
		{Name: "c", Values: []int{1, 2, 3}},
		{Name: "d", Values: []int{1, 2, 3}},
	}
}

func TestAlldiffBuildCases(t *testing.T) {
	vars := alldiffTestVars()
	tests := []struct {
		name           string
		words          []string
		wantKnown      bool
		wantMessage    string
		wantConstraint bool
		wantText       string
		wantScope      []int
		values         []int
		wantHolds      bool
	}{
		{
			name:           "three names preserve text order and sorted scope",
			words:          []string{"c", "a", "b"},
			wantKnown:      true,
			wantConstraint: true,
			wantText:       "alldiff c a b",
			wantScope:      []int{0, 1, 2},
			values:         []int{1, 2, 3, 1},
			wantHolds:      true,
		},
		{
			name:           "reverse names still return an ascending scope",
			words:          []string{"c", "a"},
			wantKnown:      true,
			wantConstraint: true,
			wantText:       "alldiff c a",
			wantScope:      []int{0, 2},
			values:         []int{1, 2, 3, 1},
			wantHolds:      true,
		},
		{
			name:           "equal values violate pairwise distinctness",
			words:          []string{"a", "b", "c"},
			wantKnown:      true,
			wantConstraint: true,
			wantText:       "alldiff a b c",
			wantScope:      []int{0, 1, 2},
			values:         []int{2, 2, 3, 1},
			wantHolds:      false,
		},
		{
			name:           "equal nonadjacent values violate pairwise distinctness",
			words:          []string{"a", "b", "c", "d"},
			wantKnown:      true,
			wantConstraint: true,
			wantText:       "alldiff a b c d",
			wantScope:      []int{0, 1, 2, 3},
			values:         []int{1, 2, 1, 3},
			wantHolds:      false,
		},
		{
			name:        "no names is invalid",
			words:       []string{},
			wantKnown:   true,
			wantMessage: "invalid alldiff statement",
		},
		{
			name:        "one name is invalid",
			words:       []string{"a"},
			wantKnown:   true,
			wantMessage: "invalid alldiff statement",
		},
		{
			name:        "unknown name precedes repeated name",
			words:       []string{"a", "ghost", "a"},
			wantKnown:   true,
			wantMessage: "unknown variable \"ghost\"",
		},
		{
			name:        "first unknown name is reported",
			words:       []string{"ghost", "missing"},
			wantKnown:   true,
			wantMessage: "unknown variable \"ghost\"",
		},
		{
			name:        "first repeated name is reported",
			words:       []string{"a", "b", "a", "b"},
			wantKnown:   true,
			wantMessage: "repeated variable \"a\"",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, msg, known := Build("alldiff", test.words, vars)
			if known != test.wantKnown {
				t.Fatalf("Build(...).known = %v, want %v", known, test.wantKnown)
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

func TestBuildUnknownKeywords(t *testing.T) {
	vars := alldiffTestVars()
	tests := []struct {
		name    string
		keyword string
		words   []string
	}{
		{name: "empty keyword", keyword: ""},
		{name: "unsupported keyword", keyword: "unknown"},
		{name: "uppercase keyword", keyword: "Alldiff", words: []string{"a", "b"}},
		{name: "variable keyword", keyword: "var", words: []string{"a", "1..2"}},
		{name: "require keyword", keyword: "require", words: []string{"a > 0"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, msg, known := Build(test.keyword, test.words, vars)
			if got != nil || msg != "" || known {
				t.Fatalf("Build(%q, %v, ...) = (%v, %q, %v), want (nil, \"\", false)", test.keyword, test.words, got, msg, known)
			}
		})
	}
}
