package lang

import (
	"strings"
	"unicode/utf8"

	"github.com/Gulliby-development/narrowfit/internal/constraint"
	"github.com/Gulliby-development/narrowfit/internal/core"
)

var reservedModelNames = map[string]struct{}{
	"var": {}, "require": {}, "alldiff": {}, "sum": {}, "count": {},
	"table": {}, "element": {}, "minimize": {}, "maximize": {},
	"and": {}, "or": {}, "not": {}, "abs": {}, "min": {}, "max": {},
	"in": {}, "is": {},
}

// Parse reads the complete model language and returns its canonical model.
func Parse(text string) (*core.Model, *core.ModelError) {
	if !utf8.ValidString(text) {
		return nil, &core.ModelError{Line: 0, Msg: "input is not valid UTF-8"}
	}
	if strings.HasPrefix(text, "\uFEFF") {
		text = text[len("\uFEFF"):]
	}

	model := &core.Model{}
	lines := strings.Split(text, "\n")
	for lineIndex, line := range lines {
		if lineIndex < len(lines)-1 && strings.HasSuffix(line, "\r") {
			line = line[:len(line)-1]
		}
		if commentAt := strings.IndexByte(line, '#'); commentAt >= 0 {
			line = line[:commentAt]
		}
		statement := strings.Trim(line, " \t")
		if statement == "" {
			continue
		}
		words := splitModelWords(statement)
		keyword := words[0]

		switch keyword {
		case "var":
			if len(words) != 3 {
				return nil, modelError(lineIndex+1, "invalid var statement")
			}
			name := words[1]
			if !isName(name) {
				return nil, modelError(lineIndex+1, "invalid name \""+name+"\"")
			}
			if _, reserved := reservedModelNames[name]; reserved {
				return nil, modelError(lineIndex+1, "reserved name \""+name+"\"")
			}
			if _, exists := core.VarIndex(model.Vars, name); exists {
				return nil, modelError(lineIndex+1, "duplicate variable \""+name+"\"")
			}
			values, ok := core.ParseDomain(words[2])
			if !ok {
				return nil, modelError(lineIndex+1, "invalid domain \""+words[2]+"\"")
			}
			if len(model.Vars) >= 100 {
				return nil, modelError(lineIndex+1, "too many variables")
			}
			model.Vars = append(model.Vars, core.Var{Name: name, Values: values})
			model.Text = append(model.Text, "var "+name+" "+core.DomainText(values))

		case "require":
			expr, msg := ParseExpr(expressionAfterKeyword(statement, keyword), model.Vars, true)
			if msg != "" {
				return nil, modelError(lineIndex+1, msg)
			}
			c := constraint.NewRequire(expr)
			model.Stmts = append(model.Stmts, core.Stmt{Line: lineIndex + 1, C: c})
			model.Text = append(model.Text, c.Text())

		case "minimize", "maximize":
			expr, msg := ParseExpr(expressionAfterKeyword(statement, keyword), model.Vars, false)
			if msg != "" {
				return nil, modelError(lineIndex+1, msg)
			}
			if model.Obj != nil {
				return nil, modelError(lineIndex+1, "duplicate objective")
			}
			model.Obj = &core.Objective{
				Line:     lineIndex + 1,
				Maximize: keyword == "maximize",
				E:        expr,
			}
			model.Text = append(model.Text, keyword+" "+core.ExprText(expr))

		case "alldiff", "sum", "count", "table", "element":
			c, msg, _ := constraint.Build(keyword, words[1:], model.Vars)
			if msg != "" {
				return nil, modelError(lineIndex+1, msg)
			}
			model.Stmts = append(model.Stmts, core.Stmt{Line: lineIndex + 1, C: c})
			model.Text = append(model.Text, c.Text())

		default:
			return nil, modelError(lineIndex+1, "unknown statement \""+keyword+"\"")
		}
	}

	if len(model.Vars) == 0 {
		return nil, &core.ModelError{Line: 0, Msg: "no variables"}
	}
	return model, nil
}

func modelError(line int, msg string) *core.ModelError {
	return &core.ModelError{Line: line, Msg: msg}
}

func splitModelWords(text string) []string {
	words := make([]string, 0)
	for i := 0; i < len(text); {
		for i < len(text) && (text[i] == ' ' || text[i] == '\t') {
			i++
		}
		if i == len(text) {
			break
		}
		start := i
		for i < len(text) && text[i] != ' ' && text[i] != '\t' {
			i++
		}
		words = append(words, text[start:i])
	}
	return words
}

func expressionAfterKeyword(statement, keyword string) string {
	expr := statement[len(keyword):]
	for len(expr) > 0 && (expr[0] == ' ' || expr[0] == '\t') {
		expr = expr[1:]
	}
	return expr
}
