// Package core defines the shared model contracts used by narrowfit.
package core

// Var is a model variable. Its index is its position in Model.Vars. Values are
// expected to be nonempty, ascending and unique.
type Var struct {
	Name   string
	Values []int
}

// Constraint is a statement that can inspect an assignment. Scope contains
// ascending, unique variable indexes and may be empty. Holds receives one value
// per model variable and reads only indexes in Scope. Text returns canonical
// statement text.
type Constraint interface {
	Scope() []int
	Holds(values []int) bool
	Text() string
}

// Stmt associates a constraint with its source line.
type Stmt struct {
	Line int
	C    Constraint
}

// Objective is the expression to minimize or maximize.
type Objective struct {
	Line     int
	Maximize bool
	E        Expr
}

// Model contains variables, constraints, an optional objective, and canonical
// text for every statement in file order, including variable and objective
// statements.
type Model struct {
	Vars  []Var
	Stmts []Stmt
	Obj   *Objective
	Text  []string
}

// ModelError reports a model error. Line is zero when the error has no source
// line; Error returns Msg unchanged.
type ModelError struct {
	Line int
	Msg  string
}

// Error returns the model error message unchanged.
func (e ModelError) Error() string { return e.Msg }

// CliError reports a command-line error; Error returns Msg unchanged.
type CliError struct {
	Msg string
}

// Error returns the command-line error message unchanged.
func (e CliError) Error() string { return e.Msg }

// VarIndex returns the declaration-position index of name in vars.
func VarIndex(vars []Var, name string) (int, bool) {
	for i := range vars {
		if vars[i].Name == name {
			return i, true
		}
	}
	return 0, false
}

// ResolveNames resolves names in input order. It reports the first unknown
// name before checking for repeated names, then reports the first repetition.
func ResolveNames(vars []Var, names []string) (indexes []int, msg string) {
	indexes = make([]int, len(names))
	for i, name := range names {
		index, ok := VarIndex(vars, name)
		if !ok {
			return nil, "unknown variable \"" + name + "\""
		}
		indexes[i] = index
	}

	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		if _, ok := seen[name]; ok {
			return nil, "repeated variable \"" + name + "\""
		}
		seen[name] = struct{}{}
	}
	return indexes, ""
}

// Compare compares a and b using one of the six supported comparison
// operators. Unsupported operators return false.
func Compare(op string, a, b int64) bool {
	switch op {
	case "==":
		return a == b
	case "!=":
		return a != b
	case "<":
		return a < b
	case "<=":
		return a <= b
	case ">":
		return a > b
	case ">=":
		return a >= b
	default:
		return false
	}
}
