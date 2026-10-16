package lang

import (
	"reflect"
	"testing"

	"github.com/Gulliby-development/narrowfit/internal/core"
)

func exprparseVars() []core.Var {
	return []core.Var{
		{Name: "x", Values: []int{-7, 0, 1, 2, 7}},
		{Name: "y", Values: []int{-2, 0, 1, 2, 3}},
		{Name: "z", Values: []int{-1, 0, 1, 3}},
		{Name: "abcdefghijklmnopqrstuvwxyzabcdef", Values: []int{0}},
	}
}

func TestParseExprValidCases(t *testing.T) {
	x := core.Ref{Index: 0, Name: "x"}
	y := core.Ref{Index: 1, Name: "y"}
	z := core.Ref{Index: 2, Name: "z"}
	longName := core.Ref{Index: 3, Name: "abcdefghijklmnopqrstuvwxyzabcdef"}
	n := func(value int64) core.Expr { return core.Num{V: value} }
	bin := func(op string, left, right core.Expr) core.Expr {
		return core.Bin{Op: op, L: left, R: right}
	}
	tests := []struct {
		name      string
		text      string
		condition bool
		want      core.Expr
		wantText  string
		wantValue int64
		wantVars  []int
		values    []int
	}{
		{name: "zero literal", text: "0", want: n(0), wantText: "0", wantValue: 0},
		{name: "upper literal boundary", text: "1000", want: n(1000), wantText: "1000", wantValue: 1000},
		{name: "single reference", text: "x", want: x, wantText: "x", wantValue: 7, wantVars: []int{0}},
		{name: "maximum length name", text: "abcdefghijklmnopqrstuvwxyzabcdef", want: longName, wantText: "abcdefghijklmnopqrstuvwxyzabcdef", wantValue: 0, wantVars: []int{3}},
		{name: "unary negative zero", text: "-0", want: core.Neg{X: n(0)}, wantText: "-0", wantValue: 0},
		{name: "unary negative reference", text: "-x", want: core.Neg{X: x}, wantText: "-x", wantValue: -7, wantVars: []int{0}},
		{name: "nested unary minus", text: "-(-x)", want: core.Neg{X: core.Neg{X: x}}, wantText: "-(-x)", wantValue: 7, wantVars: []int{0}},
		{name: "minus followed by unary minus", text: "x--2", want: bin("-", x, core.Neg{X: n(2)}), wantText: "x - -2", wantValue: 9, wantVars: []int{0}},
		{name: "spaced negative right operand", text: "3 - -2", want: bin("-", n(3), core.Neg{X: n(2)}), wantText: "3 - -2", wantValue: 5},
		{name: "tab-separated tokens", text: "x\t+\ty", want: bin("+", x, y), wantText: "x + y", wantValue: 9, wantVars: []int{0, 1}},
		{name: "multiplication binds tighter than addition", text: "x + y * z", want: bin("+", x, bin("*", y, z)), wantText: "x + y * z", wantValue: 9, wantVars: []int{0, 1, 2}},
		{name: "parentheses override multiplication precedence", text: "(x + y) * z", want: bin("*", bin("+", x, y), z), wantText: "(x + y) * z", wantValue: 9, wantVars: []int{0, 1, 2}},
		{name: "subtraction preserves a right nested tree", text: "x-(y-z)", want: bin("-", x, bin("-", y, z)), wantText: "x - (y - z)", wantValue: 6, wantVars: []int{0, 1, 2}},
		{name: "addition and subtraction share precedence", text: "x+y-z", want: bin("-", bin("+", x, y), z), wantText: "x + y - z", wantValue: 8, wantVars: []int{0, 1, 2}},
		{name: "subtraction is left associative", text: "x-y-z", want: bin("-", bin("-", x, y), z), wantText: "x - y - z", wantValue: 4, wantVars: []int{0, 1, 2}},
		{name: "division", text: "x / y", want: bin("/", x, y), wantText: "x / y", wantValue: 3, wantVars: []int{0, 1}},
		{name: "remainder", text: "x % y", want: bin("%", x, y), wantText: "x % y", wantValue: 1, wantVars: []int{0, 1}},
		{name: "negative division truncates toward zero", text: "-7 / 2", want: bin("/", core.Neg{X: n(7)}, n(2)), wantText: "-7 / 2", wantValue: -3},
		{name: "negative remainder follows the left operand", text: "-7 % 2", want: bin("%", core.Neg{X: n(7)}, n(2)), wantText: "-7 % 2", wantValue: -1},
		{name: "negative divisor keeps the left remainder sign", text: "7 % -2", want: bin("%", n(7), core.Neg{X: n(2)}), wantText: "7 % -2", wantValue: 1},
		{name: "absolute value call", text: "abs(-x)", want: core.Call{Fn: "abs", Args: []core.Expr{core.Neg{X: x}}}, wantText: "abs(-x)", wantValue: 7, wantVars: []int{0}},
		{name: "minimum call", text: "min(x, 2)", want: core.Call{Fn: "min", Args: []core.Expr{x, n(2)}}, wantText: "min(x, 2)", wantValue: 2, wantVars: []int{0}},
		{name: "maximum call with three arguments", text: "max(x, 2, 3)", want: core.Call{Fn: "max", Args: []core.Expr{x, n(2), n(3)}}, wantText: "max(x, 2, 3)", wantValue: 7, wantVars: []int{0}},
		{name: "nested calls", text: "min(abs(-x), max(y, z))", want: core.Call{Fn: "min", Args: []core.Expr{core.Call{Fn: "abs", Args: []core.Expr{core.Neg{X: x}}}, core.Call{Fn: "max", Args: []core.Expr{y, z}}}}, wantText: "min(abs(-x), max(y, z))", wantValue: 2, wantVars: []int{0, 1, 2}},
		{name: "six repeated references are legal", text: "x+x+x+x+x+x", want: bin("+", bin("+", bin("+", bin("+", bin("+", x, x), x), x), x), x), wantText: "x + x + x + x + x + x", wantValue: 42, wantVars: []int{0}},
		{name: "parenthesized atom keeps its tree", text: "((x))", want: x, wantText: "x", wantValue: 7, wantVars: []int{0}},
		{name: "six literal operands are legal", text: "0+1+2+3+4+5", want: bin("+", bin("+", bin("+", bin("+", bin("+", n(0), n(1)), n(2)), n(3)), n(4)), n(5)), wantText: "0 + 1 + 2 + 3 + 4 + 5", wantValue: 15},
		{name: "equality comparison", text: "x == y", condition: true, want: bin("==", x, y), wantText: "x == y", wantValue: 0, wantVars: []int{0, 1}},
		{name: "not equal comparison", text: "x != y", condition: true, want: bin("!=", x, y), wantText: "x != y", wantValue: 1, wantVars: []int{0, 1}},
		{name: "less than comparison", text: "x < y", condition: true, want: bin("<", x, y), wantText: "x < y", wantValue: 0, wantVars: []int{0, 1}},
		{name: "less than or equal comparison", text: "x <= y", condition: true, want: bin("<=", x, y), wantText: "x <= y", wantValue: 0, wantVars: []int{0, 1}},
		{name: "greater than comparison", text: "x > y", condition: true, want: bin(">", x, y), wantText: "x > y", wantValue: 1, wantVars: []int{0, 1}},
		{name: "greater than or equal comparison", text: "x >= y", condition: true, want: bin(">=", x, y), wantText: "x >= y", wantValue: 1, wantVars: []int{0, 1}},
		{name: "comparison accepts arithmetic operands", text: "2 < x + y", condition: true, want: bin("<", n(2), bin("+", x, y)), wantText: "2 < x + y", wantValue: 1, wantVars: []int{0, 1}},
		{name: "comparison produces a condition", text: "1 < 2", condition: true, want: bin("<", n(1), n(2)), wantText: "1 < 2", wantValue: 1},
		{name: "logical and", text: "x > 1 and y < 3", condition: true, want: bin("and", bin(">", x, n(1)), bin("<", y, n(3))), wantText: "x > 1 and y < 3", wantValue: 1, wantVars: []int{0, 1}},
		{name: "logical and false right side", text: "x > 1 and y > 3", condition: true, want: bin("and", bin(">", x, n(1)), bin(">", y, n(3))), wantText: "x > 1 and y > 3", wantValue: 0, wantVars: []int{0, 1}},
		{name: "logical and is left associative", text: "x > 1 and y > 1 and z > 0", condition: true, want: bin("and", bin("and", bin(">", x, n(1)), bin(">", y, n(1))), bin(">", z, n(0))), wantText: "x > 1 and y > 1 and z > 0", wantValue: 1, wantVars: []int{0, 1, 2}},
		{name: "logical or is left associative", text: "x > 8 or y > 1 or z > 3", condition: true, want: bin("or", bin("or", bin(">", x, n(8)), bin(">", y, n(1))), bin(">", z, n(3))), wantText: "x > 8 or y > 1 or z > 3", wantValue: 1, wantVars: []int{0, 1, 2}},
		{name: "and binds more tightly than or", text: "x > 1 or y > 3 and z > 0", condition: true, want: bin("or", bin(">", x, n(1)), bin("and", bin(">", y, n(3)), bin(">", z, n(0)))), wantText: "x > 1 or y > 3 and z > 0", wantValue: 1, wantVars: []int{0, 1, 2}},
		{name: "implication is right associative", text: "x > 1 -> y > 1 -> z > 0", condition: true, want: bin("->", bin(">", x, n(1)), bin("->", bin(">", y, n(1)), bin(">", z, n(0)))), wantText: "x > 1 -> y > 1 -> z > 0", wantValue: 1, wantVars: []int{0, 1, 2}},
		{name: "implication preserves a left nested tree", text: "(x > 1 -> y > 1) -> z > 0", condition: true, want: bin("->", bin("->", bin(">", x, n(1)), bin(">", y, n(1))), bin(">", z, n(0))), wantText: "(x > 1 -> y > 1) -> z > 0", wantValue: 1, wantVars: []int{0, 1, 2}},
		{name: "or binds more tightly than implication", text: "x > 1 or y > 1 -> z > 1", condition: true, want: bin("->", bin("or", bin(">", x, n(1)), bin(">", y, n(1))), bin(">", z, n(1))), wantText: "x > 1 or y > 1 -> z > 1", wantValue: 0, wantVars: []int{0, 1, 2}},
		{name: "division then multiplication are left associative", text: "x / y * z", want: bin("*", bin("/", x, y), z), wantText: "x / y * z", wantValue: 9, wantVars: []int{0, 1, 2}, values: []int{7, 2, 3, 0}},
		{name: "multiplication division and remainder chain left associative", text: "x * y / z % y", want: bin("%", bin("/", bin("*", x, y), z), y), wantText: "x * y / z % y", wantValue: 0, wantVars: []int{0, 1, 2}, values: []int{7, 2, 3, 0}},
		{name: "remainder chain is left associative", text: "x % y % z", want: bin("%", bin("%", x, y), z), wantText: "x % y % z", wantValue: 0, wantVars: []int{0, 1, 2}},
		{name: "not binds below comparisons", text: "not x > 1", condition: true, want: core.Not{X: bin(">", x, n(1))}, wantText: "not x > 1", wantValue: 0, wantVars: []int{0}},
		{name: "not parenthesizes conjunction", text: "not(x>1 and y<2)", condition: true, want: core.Not{X: bin("and", bin(">", x, n(1)), bin("<", y, n(2)))}, wantText: "not (x > 1 and y < 2)", wantValue: 1, wantVars: []int{0, 1}},
		{name: "repeated not prefixes", text: "not not x == 1", condition: true, want: core.Not{X: core.Not{X: bin("==", x, n(1))}}, wantText: "not not x == 1", wantValue: 0, wantVars: []int{0}},
		{name: "not comparison then conjunction", text: "not (x > 1) and y > 1", condition: true, want: bin("and", core.Not{X: bin(">", x, n(1))}, bin(">", y, n(1))), wantText: "not x > 1 and y > 1", wantValue: 0, wantVars: []int{0, 1}},
		{name: "numeric call inside comparison and or", text: "abs(x)<=max(y,2)or z==1", condition: true, want: bin("or", bin("<=", core.Call{Fn: "abs", Args: []core.Expr{x}}, core.Call{Fn: "max", Args: []core.Expr{y, n(2)}}), bin("==", z, n(1))), wantText: "abs(x) <= max(y, 2) or z == 1", wantValue: 1, wantVars: []int{0, 1, 2}},
		{name: "parenthesized logical expression", text: "(x > 1 or y > 2) and z >= 1", condition: true, want: bin("and", bin("or", bin(">", x, n(1)), bin(">", y, n(2))), bin(">=", z, n(1))), wantText: "(x > 1 or y > 2) and z >= 1", wantValue: 1, wantVars: []int{0, 1, 2}},
		{name: "implication with parenthesized antecedent", text: "not (x > 9) -> y < 3", condition: true, want: bin("->", core.Not{X: bin(">", x, n(9))}, bin("<", y, n(3))), wantText: "not x > 9 -> y < 3", wantValue: 1, wantVars: []int{0, 1}},
		{name: "numeric expression inside a call", text: "min(x + 1, max(y, z))", want: core.Call{Fn: "min", Args: []core.Expr{bin("+", x, n(1)), core.Call{Fn: "max", Args: []core.Expr{y, z}}}}, wantText: "min(x + 1, max(y, z))", wantValue: 2, wantVars: []int{0, 1, 2}},
		{name: "ExprVars sorts and deduplicates occurrences", text: "x + y * x + z", want: bin("+", bin("+", x, bin("*", y, x)), z), wantText: "x + y * x + z", wantValue: 22, wantVars: []int{0, 1, 2}},
	}

	vars := exprparseVars()
	values := []int{7, 2, 1, 0}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, msg := ParseExpr(test.text, vars, test.condition)
			if msg != "" {
				t.Fatalf("ParseExpr(%q) error = %q, want success", test.text, msg)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("ParseExpr(%q) = %#v, want %#v", test.text, got, test.want)
			}
			if text := core.ExprText(got); text != test.wantText {
				t.Errorf("ExprText(ParseExpr(%q)) = %q, want %q", test.text, text, test.wantText)
			}
			assignment := values
			if test.values != nil {
				assignment = test.values
			}
			if value, ok := core.Eval(got, assignment); !ok || value != test.wantValue {
				t.Errorf("Eval(ParseExpr(%q)) = %d, %v; want %d, true", test.text, value, ok, test.wantValue)
			}
			if indexes := core.ExprVars(got); !reflect.DeepEqual(indexes, test.wantVars) {
				t.Errorf("ExprVars(ParseExpr(%q)) = %v, want %v", test.text, indexes, test.wantVars)
			}
		})
	}
}

func TestParseExprEvaluationPropagation(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		numeric   bool
		values    []int
		wantValue int64
		wantOK    bool
	}{
		{name: "division by zero is unevaluable", text: "x / 0", numeric: true, values: []int{7, 2, 1, 0}, wantOK: false},
		{name: "remainder by zero is unevaluable", text: "x % 0", numeric: true, values: []int{7, 2, 1, 0}, wantOK: false},
		{name: "false and skips division by zero", text: "1 > 2 and x / 0 == 1", values: []int{7, 2, 1, 0}, wantValue: 0, wantOK: true},
		{name: "true or skips remainder by zero", text: "1 < 2 or x % 0 == 1", values: []int{7, 2, 1, 0}, wantValue: 1, wantOK: true},
		{name: "false implication skips division by zero", text: "1 > 2 -> x / 0 == 1", values: []int{7, 2, 1, 0}, wantValue: 1, wantOK: true},
		{name: "true and evaluates division by zero", text: "1 < 2 and x / 0 == 1", values: []int{7, 2, 1, 0}, wantOK: false},
		{name: "false or evaluates remainder by zero", text: "1 > 2 or x % 0 == 1", values: []int{7, 2, 1, 0}, wantOK: false},
		{name: "true implication evaluates division by zero", text: "1 < 2 -> x / 0 == 1", values: []int{7, 2, 1, 0}, wantOK: false},
		{name: "not propagates unevaluability", text: "not (x / 0 == 1)", values: []int{7, 2, 1, 0}, wantOK: false},
		{name: "nonzero guard safely skips division", text: "y != 0 and x / y > 1", values: []int{7, 2, 1, 0}, wantValue: 1, wantOK: true},
		{name: "zero guard safely skips division", text: "y != 0 and x / y > 1", values: []int{7, 0, 1, 0}, wantValue: 0, wantOK: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			expr, msg := ParseExpr(test.text, exprparseVars(), !test.numeric)
			if msg != "" {
				t.Fatalf("ParseExpr(%q) error = %q, want success", test.text, msg)
			}
			value, ok := core.Eval(expr, test.values)
			if value != test.wantValue || ok != test.wantOK {
				t.Errorf("Eval(ParseExpr(%q)) = %d, %v; want %d, %v", test.text, value, ok, test.wantValue, test.wantOK)
			}
		})
	}
}

func TestParseExprInvalidCases(t *testing.T) {
	vars := exprparseVars()
	tests := []struct {
		name      string
		text      string
		condition bool
		want      string
	}{
		{name: "empty expression", text: "", want: "invalid expression"},
		{name: "whitespace only", text: " \t", want: "invalid expression"},
		{name: "unexpected character", text: "x $ 1", want: "invalid expression"},
		{name: "uppercase is not a name", text: "X", want: "invalid expression"},
		{name: "underscore cannot start a name", text: "_x", want: "invalid expression"},
		{name: "name over 32 characters", text: "abcdefghijklmnopqrstuvwxyzabcdefg", want: "invalid expression"},
		{name: "unsupported non ascii character", text: "x + é", want: "invalid expression"},
		{name: "newline is not whitespace", text: "x\n+y", want: "invalid expression"},
		{name: "leading zero", text: "01", want: "invalid expression"},
		{name: "multiple-digit leading zero", text: "007", want: "invalid expression"},
		{name: "number over upper boundary", text: "1001", want: "invalid expression"},
		{name: "plus is not a sign", text: "+3", want: "invalid expression"},
		{name: "trailing operator", text: "x +", want: "invalid expression"},
		{name: "missing left operand", text: "* x", want: "invalid expression"},
		{name: "adjacent operands", text: "x y", want: "invalid expression"},
		{name: "unbalanced opening parenthesis", text: "(x + 1", want: "invalid expression"},
		{name: "unbalanced closing parenthesis", text: "x + 1)", want: "invalid expression"},
		{name: "empty parentheses", text: "()", want: "invalid expression"},
		{name: "comparison chains are rejected", text: "x < y < 3", condition: true, want: "invalid expression"},
		{name: "numeric and condition result types differ", text: "x > 1", want: "invalid expression"},
		{name: "condition required but number supplied", text: "x + 1", condition: true, want: "invalid expression"},
		{name: "logical operator rejects numeric operands", text: "x and y", condition: true, want: "invalid expression"},
		{name: "arithmetic rejects a condition operand", text: "x + (y > 1)", want: "invalid expression"},
		{name: "comparison rejects a condition operand", text: "x > (y > 1)", condition: true, want: "invalid expression"},
		{name: "call rejects a condition argument", text: "abs(x > 1)", want: "invalid expression"},
		{name: "min rejects a condition argument", text: "min(x, y > 1)", want: "invalid expression"},
		{name: "unknown function call", text: "sum(x, y)", want: "invalid expression"},
		{name: "abs requires one argument", text: "abs()", want: "invalid expression"},
		{name: "abs rejects two arguments", text: "abs(1, 2)", want: "invalid expression"},
		{name: "min requires at least two arguments", text: "min(1)", want: "invalid expression"},
		{name: "max requires at least two arguments", text: "max()", want: "invalid expression"},
		{name: "trailing function comma", text: "max(1, 2,)", want: "invalid expression"},
		{name: "comma outside call", text: "x, y", want: "invalid expression"},
		{name: "reserved logical word is not a value", text: "and", want: "invalid expression"},
		{name: "sixth character after a valid prefix is invalid", text: "x @", want: "invalid expression"},
		{name: "invalid syntax wins over an unknown name", text: "missing +", want: "invalid expression"},
		{name: "invalid result type wins over an unknown name", text: "missing + 1 < 2", want: "invalid expression"},
		{name: "invalid logical operand types win over an unknown name", text: "missing and x", condition: true, want: "invalid expression"},
		{name: "unknown name is reported after syntax validation", text: "missing + 1", want: "unknown variable \"missing\""},
		{name: "first textual unknown name wins", text: "q + missing", want: "unknown variable \"q\""},
		{name: "unknown function is syntax error before name resolution", text: "missing + sum(x, y)", want: "invalid expression"},
		{name: "unknown name wins over the operand limit", text: "missing + 1 + 2 + 3 + 4 + 5 + 6", want: "unknown variable \"missing\""},
		{name: "seven occurrences exceed the operand limit", text: "x+x+x+x+x+x+x", want: "too many operands"},
		{name: "seven literals exceed the operand limit", text: "0+1+2+3+4+5+6", want: "too many operands"},
		{name: "unknown variable in nested function is first", text: "min(missing, other)", want: "unknown variable \"missing\""},
		{name: "unknown name follows a valid prefix of max length", text: "abcdefghijklmnopqrstuvwxyzabcdeg", want: "unknown variable \"abcdefghijklmnopqrstuvwxyzabcdeg\""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, msg := ParseExpr(test.text, vars, test.condition)
			if got != nil {
				t.Errorf("ParseExpr(%q) expression = %#v, want nil", test.text, got)
			}
			if msg != test.want {
				t.Errorf("ParseExpr(%q) error = %q, want %q", test.text, msg, test.want)
			}
		})
	}
}
