package core

import (
	"sort"
	"strconv"
	"strings"
)

// Expr is an expression node.
type Expr interface{}

// Num is an integer literal.
type Num struct {
	V int64
}

// Ref refers to a variable by declaration-position Index and retains its Name
// for canonical rendering.
type Ref struct {
	Index int
	Name  string
}

// Neg is unary arithmetic negation.
type Neg struct {
	X Expr
}

// Not is logical negation.
type Not struct {
	X Expr
}

// Bin is a binary expression. Op is one of +, -, *, /, %, ==, !=, <, <=, >,
// >=, and, or, or ->.
type Bin struct {
	Op   string
	L, R Expr
}

// Call is a call to abs, min or max.
type Call struct {
	Fn   string
	Args []Expr
}

// Eval evaluates e against a model assignment. Conditions evaluate to 0 or 1.
// It returns false when an evaluated division or remainder has a zero divisor.
func Eval(e Expr, values []int) (int64, bool) {
	switch n := e.(type) {
	case Num:
		return n.V, true
	case Ref:
		if n.Index < 0 || n.Index >= len(values) {
			return 0, false
		}
		return int64(values[n.Index]), true
	case Neg:
		v, ok := Eval(n.X, values)
		if !ok {
			return 0, false
		}
		return -v, true
	case Not:
		v, ok := Eval(n.X, values)
		if !ok {
			return 0, false
		}
		if v == 0 {
			return 1, true
		}
		return 0, true
	case Bin:
		left, ok := Eval(n.L, values)
		if !ok {
			return 0, false
		}
		switch n.Op {
		case "and":
			if left == 0 {
				return 0, true
			}
			right, ok := Eval(n.R, values)
			if !ok {
				return 0, false
			}
			return truth(right), true
		case "or":
			if left != 0 {
				return 1, true
			}
			right, ok := Eval(n.R, values)
			if !ok {
				return 0, false
			}
			return truth(right), true
		case "->":
			if left == 0 {
				return 1, true
			}
			right, ok := Eval(n.R, values)
			if !ok {
				return 0, false
			}
			return truth(right), true
		}

		right, ok := Eval(n.R, values)
		if !ok {
			return 0, false
		}
		switch n.Op {
		case "+":
			return left + right, true
		case "-":
			return left - right, true
		case "*":
			return left * right, true
		case "/":
			if right == 0 {
				return 0, false
			}
			return left / right, true
		case "%":
			if right == 0 {
				return 0, false
			}
			return left % right, true
		case "==", "!=", "<", "<=", ">", ">=":
			if Compare(n.Op, left, right) {
				return 1, true
			}
			return 0, true
		default:
			return 0, false
		}
	case Call:
		if n.Fn == "abs" {
			if len(n.Args) != 1 {
				return 0, false
			}
			v, ok := Eval(n.Args[0], values)
			if !ok {
				return 0, false
			}
			if v < 0 {
				v = -v
			}
			return v, true
		}
		if (n.Fn != "min" && n.Fn != "max") || len(n.Args) < 2 {
			return 0, false
		}
		value, ok := Eval(n.Args[0], values)
		if !ok {
			return 0, false
		}
		for _, arg := range n.Args[1:] {
			next, ok := Eval(arg, values)
			if !ok {
				return 0, false
			}
			if n.Fn == "min" && next < value || n.Fn == "max" && next > value {
				value = next
			}
		}
		return value, true
	default:
		return 0, false
	}
}

// truth returns the canonical condition value for v.
func truth(v int64) int64 {
	if v == 0 {
		return 0
	}
	return 1
}

// ExprText renders e canonically while preserving its expression tree.
func ExprText(e Expr) string {
	return exprText(e)
}

func exprText(e Expr) string {
	switch n := e.(type) {
	case Num:
		return strconv.FormatInt(n.V, 10)
	case Ref:
		return n.Name
	case Neg:
		child := exprText(n.X)
		switch n.X.(type) {
		case Num, Ref, Call:
			return "-" + child
		default:
			return "-(" + child + ")"
		}
	case Not:
		child := exprText(n.X)
		if exprPrecedence(n.X) < precedenceNot {
			child = "(" + child + ")"
		}
		return "not " + child
	case Bin:
		left := exprText(n.L)
		right := exprText(n.R)
		leftPrecedence := exprPrecedence(n.L)
		rightPrecedence := exprPrecedence(n.R)
		parentPrecedence := binPrecedence(n.Op)

		if leftPrecedence < parentPrecedence ||
			leftPrecedence == parentPrecedence && n.Op == "->" {
			left = "(" + left + ")"
		}
		if rightPrecedence < parentPrecedence ||
			rightPrecedence == parentPrecedence && n.Op != "->" {
			right = "(" + right + ")"
		}
		return left + " " + n.Op + " " + right
	case Call:
		args := make([]string, len(n.Args))
		for i, arg := range n.Args {
			args[i] = exprText(arg)
		}
		return n.Fn + "(" + strings.Join(args, ", ") + ")"
	default:
		return ""
	}
}

const (
	precedenceArrow    = 1
	precedenceOr       = 2
	precedenceAnd      = 3
	precedenceNot      = 4
	precedenceCompare  = 5
	precedenceAdd      = 6
	precedenceMultiply = 7
	precedenceNeg      = 8
	precedenceAtom     = 9
)

func exprPrecedence(e Expr) int {
	switch n := e.(type) {
	case Num, Ref, Call:
		return precedenceAtom
	case Neg:
		return precedenceNeg
	case Not:
		return precedenceNot
	case Bin:
		return binPrecedence(n.Op)
	default:
		return 0
	}
}

func binPrecedence(op string) int {
	switch op {
	case "->":
		return precedenceArrow
	case "or":
		return precedenceOr
	case "and":
		return precedenceAnd
	case "==", "!=", "<", "<=", ">", ">=":
		return precedenceCompare
	case "+", "-":
		return precedenceAdd
	case "*", "/", "%":
		return precedenceMultiply
	default:
		return 0
	}
}

// ExprVars returns all referenced indexes in ascending order without
// duplicates. It traverses every branch regardless of logical short-circuiting.
func ExprVars(e Expr) []int {
	var indexes []int
	collectExprVars(e, &indexes)
	sort.Ints(indexes)
	if len(indexes) < 2 {
		return indexes
	}
	unique := indexes[:1]
	for _, index := range indexes[1:] {
		if index != unique[len(unique)-1] {
			unique = append(unique, index)
		}
	}
	return unique
}

func collectExprVars(e Expr, indexes *[]int) {
	switch n := e.(type) {
	case Ref:
		*indexes = append(*indexes, n.Index)
	case Neg:
		collectExprVars(n.X, indexes)
	case Not:
		collectExprVars(n.X, indexes)
	case Bin:
		collectExprVars(n.L, indexes)
		collectExprVars(n.R, indexes)
	case Call:
		for _, arg := range n.Args {
			collectExprVars(arg, indexes)
		}
	}
}
