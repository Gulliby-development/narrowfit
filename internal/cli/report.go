package cli

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/Gulliby-development/narrowfit/internal/core"
)

const usageText = "usage: narrowfit solve MODEL\n" +
	"       narrowfit all [--limit N] MODEL\n" +
	"       narrowfit count [--limit N] MODEL\n" +
	"       narrowfit check MODEL NAME=VALUE...\n" +
	"       narrowfit explain MODEL\n" +
	"       narrowfit fmt [--check] MODEL\n"

// writeUsage prints the standalone help text.
func writeUsage(w io.Writer) {
	_, _ = io.WriteString(w, usageText)
}

// writeDiagnostic prints one prefixed, LF-terminated command diagnostic.
func writeDiagnostic(w io.Writer, message string) {
	_, _ = fmt.Fprintf(w, "narrowfit: %s\n", message)
}

// displayPath converts a model path to its stable diagnostic form.
func displayPath(path string) string {
	if path == "-" {
		return "<stdin>"
	}
	return strings.ReplaceAll(path, "\\", "/")
}

// writeInputDiagnostic prints an input-loading diagnostic for a model path.
func writeInputDiagnostic(w io.Writer, path, message string) {
	writeDiagnostic(w, displayPath(path)+": "+message)
}

// writeModelDiagnostic prints a parser diagnostic with its source line when
// one is available.
func writeModelDiagnostic(w io.Writer, path string, modelErr *core.ModelError) {
	if modelErr == nil {
		return
	}
	location := displayPath(path)
	if modelErr.Line > 0 {
		location += ":" + strconv.Itoa(modelErr.Line)
	}
	writeDiagnostic(w, location+": "+modelErr.Msg)
}

// writeSolveResult prints the selected assignment and objective, or the
// no-solution result. It returns the command's exit code.
func writeSolveResult(w io.Writer, model *core.Model, values []int, objective int64, ok bool) int {
	if !ok || model == nil {
		_, _ = io.WriteString(w, "no solution\n")
		return 1
	}
	for i, variable := range model.Vars {
		if i >= len(values) {
			break
		}
		_, _ = fmt.Fprintf(w, "%s = %d\n", variable.Name, values[i])
	}
	if model.Obj != nil {
		_, _ = fmt.Fprintf(w, "objective = %d\n", objective)
	}
	return 0
}

// writeNumberedSolution prints one all-command assignment in declaration order.
func writeNumberedSolution(w io.Writer, number int, vars []core.Var, values []int) {
	_, _ = fmt.Fprintf(w, "%d:", number)
	for i, variable := range vars {
		if i >= len(values) {
			break
		}
		_, _ = fmt.Fprintf(w, " %s=%d", variable.Name, values[i])
	}
	_, _ = io.WriteString(w, "\n")
}

// writeSolutionSummary prints the final all/count summary line.
func writeSolutionSummary(w io.Writer, count, limit int, more bool) {
	if more {
		_, _ = fmt.Fprintf(w, "more than %d solutions\n", limit)
		return
	}
	if count == 1 {
		_, _ = io.WriteString(w, "1 solution\n")
		return
	}
	_, _ = fmt.Fprintf(w, "%d solutions\n", count)
}

// writeCheckResult prints ok or every violated statement in source order.
// It returns the command's exit code.
func writeCheckResult(w io.Writer, model *core.Model, values []int) int {
	if model == nil {
		return 1
	}
	violated := false
	for _, statement := range model.Stmts {
		if statement.C.Holds(values) {
			continue
		}
		_, _ = fmt.Fprintf(w, "violated %d: %s\n", statement.Line, statement.C.Text())
		violated = true
	}
	if violated {
		return 1
	}
	_, _ = io.WriteString(w, "ok\n")
	return 0
}

// writeExplainResult prints satisfiability or a conflict in source order.
// It returns the command's exit code.
func writeExplainResult(w io.Writer, model *core.Model, conflict []int, satisfiable bool) int {
	if satisfiable {
		_, _ = io.WriteString(w, "satisfiable\n")
		return 0
	}
	if model == nil {
		return 1
	}
	count := len(conflict)
	if count == 1 {
		_, _ = io.WriteString(w, "conflict of 1 constraint\n")
	} else {
		_, _ = fmt.Fprintf(w, "conflict of %d constraints\n", count)
	}
	for _, index := range conflict {
		if index < 0 || index >= len(model.Stmts) {
			continue
		}
		statement := model.Stmts[index]
		_, _ = fmt.Fprintf(w, "%d: %s\n", statement.Line, statement.C.Text())
	}
	return 1
}

// writeFormattedModel prints the parser's canonical model text without solving.
func writeFormattedModel(w io.Writer, model *core.Model) {
	if model == nil {
		return
	}
	for _, line := range model.Text {
		_, _ = fmt.Fprintln(w, line)
	}
}

// writeFmtCheckResult reports whether source statement lines match canonical text.
// It returns the command's exit code.
func writeFmtCheckResult(w io.Writer, path, source string, model *core.Model) int {
	if model == nil {
		return 1
	}
	if strings.HasPrefix(source, "\uFEFF") {
		source = strings.TrimPrefix(source, "\uFEFF")
	}

	lineCount := 0
	statementIndex := 0
	lines := strings.Split(source, "\n")
	for lineIndex, line := range lines {
		if lineIndex < len(lines)-1 && strings.HasSuffix(line, "\r") {
			line = line[:len(line)-1]
		}
		if commentAt := strings.IndexByte(line, '#'); commentAt >= 0 {
			line = line[:commentAt]
		}
		statementText := strings.TrimRight(line, " \t")
		if strings.Trim(statementText, " \t") == "" {
			continue
		}

		if statementText != model.Text[statementIndex] {
			_, _ = fmt.Fprintf(w, "%s:%d: not canonical, expected %s\n", displayPath(path), lineIndex+1, model.Text[statementIndex])
			lineCount++
		}
		statementIndex++
	}
	if lineCount == 0 {
		_, _ = io.WriteString(w, "ok\n")
		return 0
	}
	if lineCount == 1 {
		_, _ = io.WriteString(w, "1 line not canonical\n")
	} else {
		_, _ = fmt.Fprintf(w, "%d lines not canonical\n", lineCount)
	}
	return 1
}
