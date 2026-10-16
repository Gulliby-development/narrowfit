package core

import (
	"reflect"
	"strconv"
	"testing"
)

type coreTestConstraint struct {
	scope []int
	holds bool
	text  string
}

func (c coreTestConstraint) Scope() []int     { return c.scope }
func (c coreTestConstraint) Holds([]int) bool { return c.holds }
func (c coreTestConstraint) Text() string     { return c.text }

var (
	_ Constraint = coreTestConstraint{}
	_ error      = ModelError{}
	_ error      = CliError{}
)

func TestContractsAndErrors(t *testing.T) {
	v := Var{Name: "a", Values: []int{-1, 0, 1}}
	c := coreTestConstraint{scope: []int{0}, holds: true, text: "require a > 0"}
	stmt := Stmt{Line: 3, C: c}
	obj := Objective{Line: 4, Maximize: true, E: Ref{Index: 0, Name: "a"}}
	model := Model{
		Vars:  []Var{v},
		Stmts: []Stmt{stmt},
		Obj:   &obj,
		Text:  []string{"var a -1..1", "require a > 0", "maximize a"},
	}
	if model.Vars[0].Name != "a" || !reflect.DeepEqual(model.Vars[0].Values, []int{-1, 0, 1}) {
		t.Fatalf("Model.Vars = %#v", model.Vars)
	}
	if model.Stmts[0].Line != 3 || model.Stmts[0].C.Text() != "require a > 0" || !model.Stmts[0].C.Holds([]int{1}) {
		t.Fatalf("Model.Stmts = %#v", model.Stmts)
	}
	if !reflect.DeepEqual(model.Stmts[0].C.Scope(), []int{0}) {
		t.Fatalf("constraint scope = %v", model.Stmts[0].C.Scope())
	}
	if model.Obj == nil || model.Obj.Line != 4 || !model.Obj.Maximize || ExprText(model.Obj.E) != "a" {
		t.Fatalf("Model.Obj = %#v", model.Obj)
	}
	if !reflect.DeepEqual(model.Text, []string{"var a -1..1", "require a > 0", "maximize a"}) {
		t.Fatalf("Model.Text = %v", model.Text)
	}
	if got := (ModelError{Line: 19, Msg: " model error\n"}).Error(); got != " model error\n" {
		t.Fatalf("ModelError.Error() = %q", got)
	}
	if got := (CliError{Msg: " cli error\n"}).Error(); got != " cli error\n" {
		t.Fatalf("CliError.Error() = %q", got)
	}
}

func TestVarIndexAndResolveNames(t *testing.T) {
	vars := []Var{{Name: "a"}, {Name: "b"}, {Name: "c"}}
	if got, ok := VarIndex(vars, "b"); !ok || got != 1 {
		t.Fatalf("VarIndex(b) = %d, %v", got, ok)
	}
	if got, ok := VarIndex(vars, "q"); ok || got != 0 {
		t.Fatalf("VarIndex(q) = %d, %v", got, ok)
	}

	tests := []struct {
		name        string
		names       []string
		want        []int
		wantMessage string
	}{
		{name: "empty", names: []string{}, want: []int{}},
		{name: "input order", names: []string{"c", "a"}, want: []int{2, 0}},
		{name: "all variables", names: []string{"a", "b", "c"}, want: []int{0, 1, 2}},
		{name: "unknown before repetition", names: []string{"a", "a", "q", "r"}, wantMessage: "unknown variable \"q\""},
		{name: "first unknown", names: []string{"q", "r"}, wantMessage: "unknown variable \"q\""},
		{name: "first repeated in input order", names: []string{"b", "a", "b", "a"}, wantMessage: "repeated variable \"b\""},
		{name: "repeated singleton", names: []string{"c", "c"}, wantMessage: "repeated variable \"c\""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, message := ResolveNames(vars, test.names)
			if message != test.wantMessage {
				t.Fatalf("ResolveNames message = %q, want %q", message, test.wantMessage)
			}
			if message == "" && !reflect.DeepEqual(got, test.want) {
				t.Fatalf("ResolveNames indexes = %v, want %v", got, test.want)
			}
		})
	}
}

func TestCompare(t *testing.T) {
	tests := []struct {
		op   string
		a, b int64
		want bool
	}{
		{op: "==", a: -2, b: -2, want: true},
		{op: "==", a: -2, b: 0, want: false},
		{op: "!=", a: 0, b: 1, want: true},
		{op: "!=", a: 1, b: 1, want: false},
		{op: "<", a: -1, b: 0, want: true},
		{op: "<", a: 2, b: 1, want: false},
		{op: "<=", a: -1, b: -1, want: true},
		{op: "<=", a: 2, b: 1, want: false},
		{op: ">", a: 2, b: 1, want: true},
		{op: ">", a: 1, b: 1, want: false},
		{op: ">=", a: 2, b: 2, want: true},
		{op: ">=", a: -2, b: -1, want: false},
		{op: "<", a: -1 << 63, b: 1<<63 - 1, want: true},
		{op: ">", a: 1<<63 - 1, b: -1 << 63, want: true},
		{op: "==", a: -1 << 63, b: -1 << 63, want: true},
		{op: "==", a: 1<<63 - 1, b: -1 << 63, want: false},
		{op: "=", a: 1, b: 1, want: false},
		{op: "and", a: 1, b: 1, want: false},
		{op: "", a: 1, b: 1, want: false},
	}
	for _, test := range tests {
		if got := Compare(test.op, test.a, test.b); got != test.want {
			t.Errorf("Compare(%q, %d, %d) = %v, want %v", test.op, test.a, test.b, got, test.want)
		}
	}
}

func TestParseInt(t *testing.T) {
	tests := []struct {
		text string
		lo   int
		hi   int
		want int
		ok   bool
	}{
		{text: "0", lo: -1000, hi: 1000, want: 0, ok: true},
		{text: "3", lo: -3, hi: 3, want: 3, ok: true},
		{text: "-3", lo: -3, hi: 3, want: -3, ok: true},
		{text: "-1000", lo: -1000, hi: 1000, want: -1000, ok: true},
		{text: "1000", lo: -1000, hi: 1000, want: 1000, ok: true},
		{text: "-1000000", lo: -1000000, hi: 1000000, want: -1000000, ok: true},
		{text: "1000000", lo: -1000000, hi: 1000000, want: 1000000, ok: true},
		{text: "1", lo: 1, hi: 1, want: 1, ok: true},
		{text: "-1", lo: -1, hi: -1, want: -1, ok: true},
		{text: "", lo: -1, hi: 1, want: 0, ok: false},
		{text: "-", lo: -1, hi: 1, want: 0, ok: false},
		{text: "-0", lo: -1, hi: 1, want: 0, ok: false},
		{text: "+3", lo: -10, hi: 10, want: 0, ok: false},
		{text: "007", lo: -10, hi: 10, want: 0, ok: false},
		{text: "-007", lo: -10, hi: 10, want: 0, ok: false},
		{text: " 3", lo: -10, hi: 10, want: 0, ok: false},
		{text: "3 ", lo: -10, hi: 10, want: 0, ok: false},
		{text: "3.0", lo: -10, hi: 10, want: 0, ok: false},
		{text: "٣", lo: -10, hi: 10, want: 0, ok: false},
		{text: "1-2", lo: -10, hi: 10, want: 0, ok: false},
		{text: "1001", lo: -1000, hi: 1000, want: 0, ok: false},
		{text: "-1001", lo: -1000, hi: 1000, want: 0, ok: false},
		{text: "4", lo: 1, hi: 3, want: 0, ok: false},
		{text: "1", lo: 2, hi: 1, want: 0, ok: false},
		{text: "9223372036854775808", lo: 0, hi: int(^uint(0) >> 1), want: 0, ok: false},
		{text: "-9223372036854775809", lo: -int(^uint(0)>>1) - 1, hi: 0, want: 0, ok: false},
		{text: strconv.Itoa(int(^uint(0) >> 1)), lo: -int(^uint(0)>>1) - 1, hi: int(^uint(0) >> 1), want: int(^uint(0) >> 1), ok: true},
		{text: strconv.Itoa(-int(^uint(0)>>1) - 1), lo: -int(^uint(0)>>1) - 1, hi: int(^uint(0) >> 1), want: -int(^uint(0)>>1) - 1, ok: true},
	}
	for _, test := range tests {
		got, ok := ParseInt(test.text, test.lo, test.hi)
		if got != test.want || ok != test.ok {
			t.Errorf("ParseInt(%q, %d, %d) = %d, %v; want %d, %v", test.text, test.lo, test.hi, got, ok, test.want, test.ok)
		}
	}
}

func TestParseDomainAndCanonicalText(t *testing.T) {
	tests := []struct {
		text string
		want []int
	}{
		{text: "0", want: []int{0}},
		{text: "0,2,4", want: []int{0, 2, 4}},
		{text: "1..9", want: []int{1, 2, 3, 4, 5, 6, 7, 8, 9}},
		{text: "7,5,6,-3..3", want: []int{-3, -2, -1, 0, 1, 2, 3, 5, 6, 7}},
		{text: "2,1,2", want: []int{1, 2}},
		{text: "3..3", want: []int{3}},
		{text: "-1000,-999,1000", want: []int{-1000, -999, 1000}},
		{text: "-2,-1,0,2,4,6,7", want: []int{-2, -1, 0, 2, 4, 6, 7}},
	}
	for _, test := range tests {
		got, ok := ParseDomain(test.text)
		if !ok {
			t.Errorf("ParseDomain(%q) rejected valid input", test.text)
			continue
		}
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("ParseDomain(%q) = %v, want %v", test.text, got, test.want)
		}
		canonical := DomainText(test.want)
		roundTrip, roundTripOK := ParseDomain(canonical)
		if !roundTripOK || !reflect.DeepEqual(roundTrip, test.want) {
			t.Errorf("round trip %q = %v, %v; want %v", canonical, roundTrip, roundTripOK, test.want)
		}
	}
	whole, ok := ParseDomain("-1000..1000")
	if !ok || len(whole) != 2001 || whole[0] != -1000 || whole[1000] != 0 || whole[2000] != 1000 {
		t.Fatalf("full domain has unexpected bounds or length: len=%d ok=%v", len(whole), ok)
	}
	if got := DomainText(whole); got != "-1000..1000" {
		t.Fatalf("DomainText(full domain) = %q", got)
	}
	if got := DomainText([]int{0, 1}); got != "0..1" {
		t.Fatalf("DomainText(two-value run) = %q", got)
	}
	if got := DomainText([]int{0, 2, 4}); got != "0,2,4" {
		t.Fatalf("DomainText(singletons) = %q", got)
	}

	invalid := []string{
		"", ",1", "1,", "1,,2", "3..1", "1...2", "1..", "..1",
		"-1001", "1001", "+3", "-0", "01", "1, 2", "1 2", "1..2..3",
	}
	for _, text := range invalid {
		if got, ok := ParseDomain(text); ok {
			t.Errorf("ParseDomain(%q) = %v, true; want rejection", text, got)
		}
	}
}

func TestEvalArithmeticAndNodes(t *testing.T) {
	tests := []struct {
		name   string
		expr   Expr
		values []int
		want   int64
		ok     bool
	}{
		{name: "number", expr: Num{V: 17}, want: 17, ok: true},
		{name: "reference uses index", expr: Ref{Index: 1, Name: "wrong"}, values: []int{7, -4}, want: -4, ok: true},
		{name: "negative reference", expr: Neg{X: Ref{Index: 0, Name: "x"}}, values: []int{7}, want: -7, ok: true},
		{name: "not zero", expr: Not{X: Num{V: 0}}, want: 1, ok: true},
		{name: "not nonzero", expr: Not{X: Num{V: -7}}, want: 0, ok: true},
		{name: "add", expr: Bin{Op: "+", L: Num{V: -3}, R: Num{V: 8}}, want: 5, ok: true},
		{name: "subtract", expr: Bin{Op: "-", L: Num{V: 3}, R: Num{V: 8}}, want: -5, ok: true},
		{name: "multiply", expr: Bin{Op: "*", L: Num{V: -3}, R: Num{V: 8}}, want: -24, ok: true},
		{name: "divide truncates negative dividend", expr: Bin{Op: "/", L: Num{V: -7}, R: Num{V: 2}}, want: -3, ok: true},
		{name: "divide truncates negative divisor", expr: Bin{Op: "/", L: Num{V: 7}, R: Num{V: -2}}, want: -3, ok: true},
		{name: "remainder follows negative dividend", expr: Bin{Op: "%", L: Num{V: -7}, R: Num{V: 2}}, want: -1, ok: true},
		{name: "remainder follows negative divisor", expr: Bin{Op: "%", L: Num{V: 7}, R: Num{V: -2}}, want: 1, ok: true},
		{name: "abs negative", expr: Call{Fn: "abs", Args: []Expr{Num{V: -7}}}, want: 7, ok: true},
		{name: "abs nonnegative", expr: Call{Fn: "abs", Args: []Expr{Num{V: 7}}}, want: 7, ok: true},
		{name: "min", expr: Call{Fn: "min", Args: []Expr{Num{V: 3}, Num{V: -2}, Num{V: 1}}}, want: -2, ok: true},
		{name: "max", expr: Call{Fn: "max", Args: []Expr{Num{V: 3}, Num{V: -2}, Num{V: 1}}}, want: 3, ok: true},
		{name: "missing reference", expr: Ref{Index: 1, Name: "b"}, values: []int{1}, want: 0, ok: false},
		{name: "negative reference index", expr: Ref{Index: -1, Name: "a"}, values: []int{1}, want: 0, ok: false},
		{name: "unknown operator", expr: Bin{Op: "xor", L: Num{V: 1}, R: Num{V: 2}}, want: 0, ok: false},
		{name: "unknown function", expr: Call{Fn: "sum", Args: []Expr{Num{V: 1}, Num{V: 2}}}, want: 0, ok: false},
		{name: "abs wrong arity", expr: Call{Fn: "abs", Args: []Expr{Num{V: 1}, Num{V: 2}}}, want: 0, ok: false},
		{name: "min wrong arity", expr: Call{Fn: "min", Args: []Expr{Num{V: 1}}}, want: 0, ok: false},
	}
	for _, test := range tests {
		got, ok := Eval(test.expr, test.values)
		if got != test.want || ok != test.ok {
			t.Errorf("%s: Eval() = %d, %v; want %d, %v", test.name, got, ok, test.want, test.ok)
		}
	}
}

func TestEvalComparisonsAndLogicalOperators(t *testing.T) {
	tests := []struct {
		name string
		expr Expr
		want int64
		ok   bool
	}{
		{name: "equal true", expr: Bin{Op: "==", L: Num{V: 2}, R: Num{V: 2}}, want: 1, ok: true},
		{name: "equal false", expr: Bin{Op: "==", L: Num{V: 2}, R: Num{V: 3}}, want: 0, ok: true},
		{name: "not equal true", expr: Bin{Op: "!=", L: Num{V: 2}, R: Num{V: 3}}, want: 1, ok: true},
		{name: "not equal false", expr: Bin{Op: "!=", L: Num{V: 2}, R: Num{V: 2}}, want: 0, ok: true},
		{name: "less true", expr: Bin{Op: "<", L: Num{V: 2}, R: Num{V: 3}}, want: 1, ok: true},
		{name: "less false", expr: Bin{Op: "<", L: Num{V: 3}, R: Num{V: 2}}, want: 0, ok: true},
		{name: "less equal true", expr: Bin{Op: "<=", L: Num{V: 2}, R: Num{V: 2}}, want: 1, ok: true},
		{name: "less equal false", expr: Bin{Op: "<=", L: Num{V: 3}, R: Num{V: 2}}, want: 0, ok: true},
		{name: "greater true", expr: Bin{Op: ">", L: Num{V: 3}, R: Num{V: 2}}, want: 1, ok: true},
		{name: "greater false", expr: Bin{Op: ">", L: Num{V: 2}, R: Num{V: 3}}, want: 0, ok: true},
		{name: "greater equal true", expr: Bin{Op: ">=", L: Num{V: 2}, R: Num{V: 2}}, want: 1, ok: true},
		{name: "greater equal false", expr: Bin{Op: ">=", L: Num{V: 2}, R: Num{V: 3}}, want: 0, ok: true},
		{name: "and false left", expr: Bin{Op: "and", L: Num{V: 0}, R: Num{V: 1}}, want: 0, ok: true},
		{name: "and false right", expr: Bin{Op: "and", L: Num{V: 1}, R: Num{V: 0}}, want: 0, ok: true},
		{name: "and true", expr: Bin{Op: "and", L: Num{V: -1}, R: Num{V: 2}}, want: 1, ok: true},
		{name: "or false", expr: Bin{Op: "or", L: Num{V: 0}, R: Num{V: 0}}, want: 0, ok: true},
		{name: "or right true", expr: Bin{Op: "or", L: Num{V: 0}, R: Num{V: -2}}, want: 1, ok: true},
		{name: "or left true", expr: Bin{Op: "or", L: Num{V: 2}, R: Num{V: 0}}, want: 1, ok: true},
		{name: "arrow false antecedent", expr: Bin{Op: "->", L: Num{V: 0}, R: Num{V: 0}}, want: 1, ok: true},
		{name: "arrow true consequent", expr: Bin{Op: "->", L: Num{V: 1}, R: Num{V: -2}}, want: 1, ok: true},
		{name: "arrow false consequent", expr: Bin{Op: "->", L: Num{V: 1}, R: Num{V: 0}}, want: 0, ok: true},
		{name: "logical operands normalize truth", expr: Bin{Op: "and", L: Num{V: 5}, R: Num{V: -9}}, want: 1, ok: true},
	}
	for _, test := range tests {
		got, ok := Eval(test.expr, nil)
		if got != test.want || ok != test.ok {
			t.Errorf("%s: Eval() = %d, %v; want %d, %v", test.name, got, ok, test.want, test.ok)
		}
	}
}

func TestEvalShortCircuitAndFailurePropagation(t *testing.T) {
	badDivision := Bin{Op: "/", L: Num{V: 1}, R: Num{V: 0}}
	badRemainder := Bin{Op: "%", L: Num{V: 1}, R: Num{V: 0}}
	tests := []struct {
		name string
		expr Expr
		want int64
		ok   bool
	}{
		{name: "false and skips division", expr: Bin{Op: "and", L: Num{V: 0}, R: badDivision}, want: 0, ok: true},
		{name: "true or skips division", expr: Bin{Op: "or", L: Num{V: 1}, R: badDivision}, want: 1, ok: true},
		{name: "false implication skips remainder", expr: Bin{Op: "->", L: Num{V: 0}, R: badRemainder}, want: 1, ok: true},
		{name: "true and evaluates division", expr: Bin{Op: "and", L: Num{V: 1}, R: badDivision}, want: 0, ok: false},
		{name: "false or evaluates division", expr: Bin{Op: "or", L: Num{V: 0}, R: badDivision}, want: 0, ok: false},
		{name: "true implication evaluates division", expr: Bin{Op: "->", L: Num{V: 1}, R: badDivision}, want: 0, ok: false},
		{name: "not propagates division failure", expr: Not{X: badDivision}, want: 0, ok: false},
		{name: "negation propagates remainder failure", expr: Neg{X: badRemainder}, want: 0, ok: false},
		{name: "addition propagates failure", expr: Bin{Op: "+", L: Num{V: 4}, R: badDivision}, want: 0, ok: false},
		{name: "comparison propagates failure", expr: Bin{Op: "==", L: badDivision, R: Num{V: 0}}, want: 0, ok: false},
		{name: "abs propagates failure", expr: Call{Fn: "abs", Args: []Expr{badDivision}}, want: 0, ok: false},
		{name: "min propagates failure", expr: Call{Fn: "min", Args: []Expr{Num{V: 1}, badDivision}}, want: 0, ok: false},
		{name: "max propagates failure", expr: Call{Fn: "max", Args: []Expr{badRemainder, Num{V: 1}}}, want: 0, ok: false},
		{name: "left-side failure is not skipped", expr: Bin{Op: "and", L: badDivision, R: Num{V: 0}}, want: 0, ok: false},
	}
	for _, test := range tests {
		got, ok := Eval(test.expr, nil)
		if got != test.want || ok != test.ok {
			t.Errorf("%s: Eval() = %d, %v; want %d, %v", test.name, got, ok, test.want, test.ok)
		}
	}
}

func TestExprTextPrecedenceAndTreePreservation(t *testing.T) {
	tests := []struct {
		name string
		expr Expr
		want string
	}{
		{name: "number", expr: Num{V: -12}, want: "-12"},
		{name: "reference", expr: Ref{Index: 0, Name: "x"}, want: "x"},
		{name: "all binary operator spacing", expr: Bin{Op: "+", L: Ref{Index: 0, Name: "x"}, R: Num{V: 2}}, want: "x + 2"},
		{name: "subtraction", expr: Bin{Op: "-", L: Ref{Index: 0, Name: "x"}, R: Num{V: 2}}, want: "x - 2"},
		{name: "multiplication", expr: Bin{Op: "*", L: Ref{Index: 0, Name: "x"}, R: Num{V: 2}}, want: "x * 2"},
		{name: "division", expr: Bin{Op: "/", L: Ref{Index: 0, Name: "x"}, R: Num{V: 2}}, want: "x / 2"},
		{name: "remainder", expr: Bin{Op: "%", L: Ref{Index: 0, Name: "x"}, R: Num{V: 2}}, want: "x % 2"},
		{name: "equality", expr: Bin{Op: "==", L: Ref{Index: 0, Name: "x"}, R: Num{V: 2}}, want: "x == 2"},
		{name: "inequality", expr: Bin{Op: "!=", L: Ref{Index: 0, Name: "x"}, R: Num{V: 2}}, want: "x != 2"},
		{name: "less", expr: Bin{Op: "<", L: Ref{Index: 0, Name: "x"}, R: Num{V: 2}}, want: "x < 2"},
		{name: "less equal", expr: Bin{Op: "<=", L: Ref{Index: 0, Name: "x"}, R: Num{V: 2}}, want: "x <= 2"},
		{name: "greater", expr: Bin{Op: ">", L: Ref{Index: 0, Name: "x"}, R: Num{V: 2}}, want: "x > 2"},
		{name: "greater equal", expr: Bin{Op: ">=", L: Ref{Index: 0, Name: "x"}, R: Num{V: 2}}, want: "x >= 2"},
		{name: "and", expr: Bin{Op: "and", L: Ref{Index: 0, Name: "x"}, R: Ref{Index: 1, Name: "y"}}, want: "x and y"},
		{name: "or", expr: Bin{Op: "or", L: Ref{Index: 0, Name: "x"}, R: Ref{Index: 1, Name: "y"}}, want: "x or y"},
		{name: "implication", expr: Bin{Op: "->", L: Ref{Index: 0, Name: "x"}, R: Ref{Index: 1, Name: "y"}}, want: "x -> y"},
		{name: "right subtraction", expr: Bin{Op: "-", L: Ref{Index: 0, Name: "x"}, R: Bin{Op: "-", L: Ref{Index: 1, Name: "y"}, R: Ref{Index: 2, Name: "z"}}}, want: "x - (y - z)"},
		{name: "left addition no parentheses", expr: Bin{Op: "+", L: Bin{Op: "+", L: Ref{Index: 0, Name: "x"}, R: Ref{Index: 1, Name: "y"}}, R: Ref{Index: 2, Name: "z"}}, want: "x + y + z"},
		{name: "right addition parentheses", expr: Bin{Op: "+", L: Ref{Index: 0, Name: "x"}, R: Bin{Op: "+", L: Ref{Index: 1, Name: "y"}, R: Ref{Index: 2, Name: "z"}}}, want: "x + (y + z)"},
		{name: "left division no parentheses", expr: Bin{Op: "/", L: Bin{Op: "/", L: Ref{Index: 0, Name: "x"}, R: Ref{Index: 1, Name: "y"}}, R: Ref{Index: 2, Name: "z"}}, want: "x / y / z"},
		{name: "right multiplication parenthesized", expr: Bin{Op: "*", L: Ref{Index: 0, Name: "x"}, R: Bin{Op: "*", L: Ref{Index: 1, Name: "y"}, R: Ref{Index: 2, Name: "z"}}}, want: "x * (y * z)"},
		{name: "weaker left child", expr: Bin{Op: "*", L: Bin{Op: "+", L: Ref{Index: 0, Name: "x"}, R: Ref{Index: 1, Name: "y"}}, R: Ref{Index: 2, Name: "z"}}, want: "(x + y) * z"},
		{name: "weaker right child", expr: Bin{Op: "+", L: Ref{Index: 0, Name: "x"}, R: Bin{Op: "*", L: Ref{Index: 1, Name: "y"}, R: Ref{Index: 2, Name: "z"}}}, want: "x + y * z"},
		{name: "comparison around arithmetic", expr: Bin{Op: "<", L: Bin{Op: "+", L: Ref{Index: 0, Name: "x"}, R: Num{V: 1}}, R: Bin{Op: "*", L: Ref{Index: 1, Name: "y"}, R: Num{V: 2}}}, want: "x + 1 < y * 2"},
		{name: "logical and inside or", expr: Bin{Op: "or", L: Bin{Op: "and", L: Ref{Index: 0, Name: "a"}, R: Ref{Index: 1, Name: "b"}}, R: Ref{Index: 2, Name: "c"}}, want: "a and b or c"},
		{name: "same-precedence right logical child", expr: Bin{Op: "and", L: Ref{Index: 0, Name: "a"}, R: Bin{Op: "and", L: Ref{Index: 1, Name: "b"}, R: Ref{Index: 2, Name: "c"}}}, want: "a and (b and c)"},
		{name: "left implication child", expr: Bin{Op: "->", L: Bin{Op: "->", L: Bin{Op: ">", L: Ref{Index: 0, Name: "x"}, R: Num{V: 1}}, R: Bin{Op: ">", L: Ref{Index: 1, Name: "y"}, R: Num{V: 1}}}, R: Bin{Op: ">", L: Ref{Index: 2, Name: "z"}, R: Num{V: 0}}}, want: "(x > 1 -> y > 1) -> z > 0"},
		{name: "right implication child", expr: Bin{Op: "->", L: Bin{Op: ">", L: Ref{Index: 0, Name: "x"}, R: Num{V: 1}}, R: Bin{Op: "->", L: Bin{Op: ">", L: Ref{Index: 1, Name: "y"}, R: Num{V: 1}}, R: Bin{Op: ">", L: Ref{Index: 2, Name: "z"}, R: Num{V: 0}}}}, want: "x > 1 -> y > 1 -> z > 0"},
		{name: "or around implication", expr: Bin{Op: "->", L: Bin{Op: "or", L: Ref{Index: 0, Name: "a"}, R: Ref{Index: 1, Name: "b"}}, R: Ref{Index: 2, Name: "c"}}, want: "a or b -> c"},
		{name: "not conjunction", expr: Not{X: Bin{Op: "and", L: Ref{Index: 0, Name: "a"}, R: Ref{Index: 1, Name: "b"}}}, want: "not (a and b)"},
		{name: "not comparison", expr: Not{X: Bin{Op: ">", L: Ref{Index: 0, Name: "x"}, R: Num{V: 1}}}, want: "not x > 1"},
		{name: "nested negation", expr: Neg{X: Neg{X: Ref{Index: 0, Name: "x"}}}, want: "-(-x)"},
		{name: "negated sum", expr: Neg{X: Bin{Op: "+", L: Ref{Index: 0, Name: "a"}, R: Ref{Index: 1, Name: "b"}}}, want: "-(a + b)"},
		{name: "negative right operand", expr: Bin{Op: "-", L: Num{V: 3}, R: Neg{X: Num{V: 2}}}, want: "3 - -2"},
		{name: "negative number", expr: Neg{X: Num{V: 2}}, want: "-2"},
		{name: "negative call", expr: Neg{X: Call{Fn: "abs", Args: []Expr{Ref{Index: 0, Name: "x"}}}}, want: "-abs(x)"},
		{name: "call arguments", expr: Call{Fn: "min", Args: []Expr{Ref{Index: 0, Name: "a"}, Num{V: 2}, Ref{Index: 1, Name: "b"}}}, want: "min(a, 2, b)"},
		{name: "nested calls and comparison", expr: Bin{Op: "<=", L: Call{Fn: "abs", Args: []Expr{Ref{Index: 0, Name: "x"}}}, R: Bin{Op: "or", L: Call{Fn: "max", Args: []Expr{Ref{Index: 1, Name: "y"}, Num{V: 2}}}, R: Bin{Op: "==", L: Ref{Index: 2, Name: "z"}, R: Num{V: 1}}}}, want: "abs(x) <= (max(y, 2) or z == 1)"},
		{name: "product remainder", expr: Bin{Op: "!=", L: Bin{Op: "%", L: Bin{Op: "*", L: Ref{Index: 0, Name: "x"}, R: Bin{Op: "+", L: Ref{Index: 1, Name: "y"}, R: Ref{Index: 2, Name: "z"}}}, R: Num{V: 2}}, R: Num{V: 0}}, want: "x * (y + z) % 2 != 0"},
	}
	for _, test := range tests {
		if got := ExprText(test.expr); got != test.want {
			t.Errorf("%s: ExprText() = %q, want %q", test.name, got, test.want)
		}
	}
}

func TestExprVars(t *testing.T) {
	if got := ExprVars(Num{V: 3}); len(got) != 0 {
		t.Fatalf("ExprVars(constant) = %v, want empty", got)
	}
	expr := Bin{
		Op: "and",
		L:  Bin{Op: "==", L: Ref{Index: 2, Name: "c"}, R: Ref{Index: 0, Name: "a"}},
		R:  Call{Fn: "max", Args: []Expr{Ref{Index: 2, Name: "c"}, Ref{Index: 1, Name: "b"}}},
	}
	if got := ExprVars(expr); !reflect.DeepEqual(got, []int{0, 1, 2}) {
		t.Fatalf("ExprVars(repeated mixed tree) = %v", got)
	}
	shortCircuitTree := Bin{
		Op: "or",
		L:  Num{V: 1},
		R:  Bin{Op: "==", L: Ref{Index: 4, Name: "skipped"}, R: Num{V: 0}},
	}
	if got := ExprVars(shortCircuitTree); !reflect.DeepEqual(got, []int{4}) {
		t.Fatalf("ExprVars(skipped branch) = %v", got)
	}
	unaryAndNestedCalls := Neg{X: Call{Fn: "min", Args: []Expr{
		Ref{Index: 5, Name: "f"},
		Not{X: Bin{Op: "==", L: Ref{Index: 3, Name: "d"}, R: Ref{Index: 5, Name: "f"}}},
	}}}
	if got := ExprVars(unaryAndNestedCalls); !reflect.DeepEqual(got, []int{3, 5}) {
		t.Fatalf("ExprVars(unary/call tree) = %v", got)
	}
}
