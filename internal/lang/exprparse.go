package lang

import (
	"github.com/Gulliby-development/narrowfit/internal/core"
)

type expressionTokenKind uint8

const (
	expressionNumber expressionTokenKind = iota
	expressionName
	expressionOperatorToken
	expressionLeftParen
	expressionRightParen
	expressionComma
)

type expressionToken struct {
	kind expressionTokenKind
	text string
	num  int
}

type expressionValue struct {
	expr      core.Expr
	condition bool
}

type expressionOperatorKind uint8

const (
	expressionBinary expressionOperatorKind = iota
	expressionNegation
	expressionNot
	expressionOpenParen
)

type expressionOperator struct {
	kind      expressionOperatorKind
	text      string
	base      int
	function  string
	completed int
}

type expressionReference struct {
	name  string
	index int
	known bool
}

// ParseExpr parses a typed numeric expression or condition. Syntax and typing
// are checked before names are resolved and before the operand limit is applied.
func ParseExpr(text string, vars []core.Var, condition bool) (core.Expr, string) {
	tokens, ok := lexExpression(text)
	if !ok {
		return nil, "invalid expression"
	}

	values := make([]expressionValue, 0, len(tokens))
	operators := make([]expressionOperator, 0, len(tokens))
	references := make([]expressionReference, 0)
	operandCount := 0
	expectOperand := true

	for i := 0; i < len(tokens); i++ {
		token := tokens[i]
		switch token.kind {
		case expressionNumber:
			if !expectOperand {
				return nil, "invalid expression"
			}
			values = append(values, expressionValue{expr: core.Num{V: int64(token.num)}})
			operandCount++
			expectOperand = false

		case expressionName:
			if !expectOperand {
				return nil, "invalid expression"
			}
			if i+1 < len(tokens) && tokens[i+1].kind == expressionLeftParen {
				if token.text != "abs" && token.text != "min" && token.text != "max" {
					return nil, "invalid expression"
				}
				operators = append(operators, expressionOperator{
					kind:     expressionOpenParen,
					base:     len(values),
					function: token.text,
				})
				i++
				expectOperand = true
				continue
			}

			index, known := core.VarIndex(vars, token.text)
			references = append(references, expressionReference{
				name: token.text, index: index, known: known,
			})
			values = append(values, expressionValue{
				expr: core.Ref{Index: index, Name: token.text},
			})
			operandCount++
			expectOperand = false

		case expressionLeftParen:
			if !expectOperand {
				return nil, "invalid expression"
			}
			operators = append(operators, expressionOperator{
				kind: expressionOpenParen,
				base: len(values),
			})
			expectOperand = true

		case expressionRightParen:
			if expectOperand {
				return nil, "invalid expression"
			}
			if !reduceUntilOpenParen(&operators, &values) {
				return nil, "invalid expression"
			}
			open := operators[len(operators)-1]
			if len(values) != open.base+open.completed+1 {
				return nil, "invalid expression"
			}
			operators = operators[:len(operators)-1]

			if open.function != "" {
				argumentCount := open.completed + 1
				if open.function == "abs" && argumentCount != 1 ||
					(open.function == "min" || open.function == "max") && argumentCount < 2 {
					return nil, "invalid expression"
				}
				arguments := make([]core.Expr, argumentCount)
				for arg := 0; arg < argumentCount; arg++ {
					value := values[open.base+arg]
					if value.condition {
						return nil, "invalid expression"
					}
					arguments[arg] = value.expr
				}
				values = values[:open.base]
				values = append(values, expressionValue{
					expr: core.Call{Fn: open.function, Args: arguments},
				})
			}
			expectOperand = false

		case expressionComma:
			if expectOperand || !reduceUntilOpenParen(&operators, &values) {
				return nil, "invalid expression"
			}
			open := &operators[len(operators)-1]
			if open.function == "" || len(values) != open.base+open.completed+1 {
				return nil, "invalid expression"
			}
			if values[len(values)-1].condition {
				return nil, "invalid expression"
			}
			open.completed++
			expectOperand = true

		case expressionOperatorToken:
			if token.text == "-" && expectOperand {
				operators = append(operators, expressionOperator{kind: expressionNegation})
				continue
			}
			if token.text == "not" && expectOperand {
				operators = append(operators, expressionOperator{kind: expressionNot})
				continue
			}
			if token.text == "and" || token.text == "or" || token.text == "->" ||
				token.text == "+" || token.text == "-" || token.text == "*" ||
				token.text == "/" || token.text == "%" || token.text == "==" ||
				token.text == "!=" || token.text == "<" || token.text == "<=" ||
				token.text == ">" || token.text == ">=" {
				if expectOperand {
					return nil, "invalid expression"
				}
				if !reduceForIncoming(token.text, &operators, &values) {
					return nil, "invalid expression"
				}
				operators = append(operators, expressionOperator{
					kind: expressionBinary, text: token.text,
				})
				expectOperand = true
				continue
			}
			return nil, "invalid expression"
		}
	}

	if expectOperand {
		return nil, "invalid expression"
	}
	for len(operators) > 0 {
		if operators[len(operators)-1].kind == expressionOpenParen ||
			!reduceExpressionOperator(&operators, &values) {
			return nil, "invalid expression"
		}
	}
	if len(values) != 1 || values[0].condition != condition {
		return nil, "invalid expression"
	}

	for _, reference := range references {
		if !reference.known {
			return nil, "unknown variable \"" + reference.name + "\""
		}
	}
	if operandCount > 6 {
		return nil, "too many operands"
	}
	return values[0].expr, ""
}

func lexExpression(text string) ([]expressionToken, bool) {
	tokens := make([]expressionToken, 0)
	for i := 0; i < len(text); {
		ch := text[i]
		if ch == ' ' || ch == '\t' {
			i++
			continue
		}
		if ch >= '0' && ch <= '9' {
			start := i
			for i < len(text) && text[i] >= '0' && text[i] <= '9' {
				i++
			}
			word := text[start:i]
			number, ok := core.ParseInt(word, 0, 1000)
			if !ok {
				return nil, false
			}
			tokens = append(tokens, expressionToken{
				kind: expressionNumber, text: word, num: number,
			})
			continue
		}
		if ch >= 'a' && ch <= 'z' {
			start := i
			i++
			for i < len(text) && isNameTail(text[i]) {
				i++
			}
			word := text[start:i]
			if !isName(word) {
				return nil, false
			}
			switch word {
			case "and", "or", "not":
				tokens = append(tokens, expressionToken{
					kind: expressionOperatorToken, text: word,
				})
			default:
				tokens = append(tokens, expressionToken{
					kind: expressionName, text: word,
				})
			}
			continue
		}

		switch ch {
		case '+', '*', '/', '%':
			tokens = append(tokens, expressionToken{
				kind: expressionOperatorToken, text: text[i : i+1],
			})
			i++
		case '(':
			tokens = append(tokens, expressionToken{kind: expressionLeftParen, text: "("})
			i++
		case ')':
			tokens = append(tokens, expressionToken{kind: expressionRightParen, text: ")"})
			i++
		case ',':
			tokens = append(tokens, expressionToken{kind: expressionComma, text: ","})
			i++
		case '=':
			if i+1 >= len(text) || text[i+1] != '=' {
				return nil, false
			}
			tokens = append(tokens, expressionToken{kind: expressionOperatorToken, text: "=="})
			i += 2
		case '!':
			if i+1 >= len(text) || text[i+1] != '=' {
				return nil, false
			}
			tokens = append(tokens, expressionToken{kind: expressionOperatorToken, text: "!="})
			i += 2
		case '<', '>':
			op := text[i : i+1]
			i++
			if i < len(text) && text[i] == '=' {
				op += "="
				i++
			}
			tokens = append(tokens, expressionToken{kind: expressionOperatorToken, text: op})
		case '-':
			if i+1 < len(text) && text[i+1] == '>' {
				tokens = append(tokens, expressionToken{kind: expressionOperatorToken, text: "->"})
				i += 2
			} else {
				tokens = append(tokens, expressionToken{kind: expressionOperatorToken, text: "-"})
				i++
			}
		default:
			return nil, false
		}
	}
	return tokens, true
}

func reduceUntilOpenParen(operators *[]expressionOperator, values *[]expressionValue) bool {
	for len(*operators) > 0 {
		if (*operators)[len(*operators)-1].kind == expressionOpenParen {
			return true
		}
		if !reduceExpressionOperator(operators, values) {
			return false
		}
	}
	return false
}

func reduceForIncoming(op string, operators *[]expressionOperator, values *[]expressionValue) bool {
	incomingPrecedence := binaryExpressionPrecedence(op)
	rightAssociative := op == "->"
	for len(*operators) > 0 {
		top := (*operators)[len(*operators)-1]
		if top.kind == expressionOpenParen {
			break
		}
		topPrecedence := expressionOperatorPrecedence(top)
		if topPrecedence < incomingPrecedence ||
			topPrecedence == incomingPrecedence && rightAssociative {
			break
		}
		if !reduceExpressionOperator(operators, values) {
			return false
		}
	}
	return true
}

func reduceExpressionOperator(operators *[]expressionOperator, values *[]expressionValue) bool {
	if len(*operators) == 0 {
		return false
	}
	last := len(*operators) - 1
	op := (*operators)[last]
	if op.kind == expressionOpenParen {
		return false
	}
	*operators = (*operators)[:last]

	switch op.kind {
	case expressionNegation, expressionNot:
		if len(*values) < 1 {
			return false
		}
		value := (*values)[len(*values)-1]
		if op.kind == expressionNegation {
			if value.condition {
				return false
			}
			(*values)[len(*values)-1] = expressionValue{expr: core.Neg{X: value.expr}}
		} else {
			if !value.condition {
				return false
			}
			(*values)[len(*values)-1] = expressionValue{
				expr: core.Not{X: value.expr}, condition: true,
			}
		}
		return true

	case expressionBinary:
		if len(*values) < 2 {
			return false
		}
		right := (*values)[len(*values)-1]
		left := (*values)[len(*values)-2]
		isLogical := op.text == "and" || op.text == "or" || op.text == "->"
		isComparison := op.text == "==" || op.text == "!=" || op.text == "<" ||
			op.text == "<=" || op.text == ">" || op.text == ">="
		if isLogical {
			if !left.condition || !right.condition {
				return false
			}
		} else if left.condition || right.condition {
			return false
		}
		*values = (*values)[:len(*values)-2]
		*values = append(*values, expressionValue{
			expr:      core.Bin{Op: op.text, L: left.expr, R: right.expr},
			condition: isLogical || isComparison,
		})
		return true
	}
	return false
}

func expressionOperatorPrecedence(op expressionOperator) int {
	switch op.kind {
	case expressionNegation:
		return 8
	case expressionNot:
		return 4
	case expressionBinary:
		return binaryExpressionPrecedence(op.text)
	default:
		return 0
	}
}

func binaryExpressionPrecedence(op string) int {
	switch op {
	case "->":
		return 1
	case "or":
		return 2
	case "and":
		return 3
	case "==", "!=", "<", "<=", ">", ">=":
		return 5
	case "+", "-":
		return 6
	case "*", "/", "%":
		return 7
	default:
		return 0
	}
}

func lexNameStart(ch byte) bool {
	return ch >= 'a' && ch <= 'z'
}

func isNameTail(ch byte) bool {
	return lexNameStart(ch) || ch >= '0' && ch <= '9' || ch == '_'
}

// isName reports whether word is a NAME in the model-language grammar.
func isName(word string) bool {
	if len(word) < 1 || len(word) > 32 || !lexNameStart(word[0]) {
		return false
	}
	for i := 1; i < len(word); i++ {
		if !isNameTail(word[i]) {
			return false
		}
	}
	return true
}
