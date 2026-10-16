package cli

import (
	"io"

	"github.com/Gulliby-development/narrowfit/internal/core"
	"github.com/Gulliby-development/narrowfit/internal/lang"
	"github.com/Gulliby-development/narrowfit/internal/solve"
)

// Run validates arguments, loads a model through the supplied providers, and
// executes the selected command.
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer, readFile func(path string) ([]byte, error)) int {
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}

	parsed, help, err := parseArgs(args)
	if err != nil {
		writeDiagnostic(stderr, err.Error())
		return 2
	}
	if help {
		writeUsage(stdout)
		return 0
	}

	model, ok := loadModel(parsed.modelPath, stdin, readFile, stderr)
	if !ok {
		return 2
	}

	switch parsed.command {
	case "solve":
		values, objective, found := solve.Best(model)
		return writeSolveResult(stdout, model, values, objective, found)
	case "all":
		number := 0
		count, more := solve.Count(model, parsed.limit, func(values []int) {
			number++
			writeNumberedSolution(stdout, number, model.Vars, values)
		})
		writeSolutionSummary(stdout, count, parsed.limit, more)
		if count == 0 {
			return 1
		}
		return 0
	case "count":
		count, more := solve.Count(model, parsed.limit, nil)
		writeSolutionSummary(stdout, count, parsed.limit, more)
		if count == 0 {
			return 1
		}
		return 0
	case "check":
		values, err := checkAssignment(model, parsed.assignments)
		if err != nil {
			writeDiagnostic(stderr, err.Error())
			return 2
		}
		return writeCheckResult(stdout, model, values)
	case "explain":
		conflict, satisfiable := solve.Explain(model)
		return writeExplainResult(stdout, model, conflict, satisfiable)
	case "fmt":
		writeFormattedModel(stdout, model)
		return 0
	default:
		// parseArgs only returns supported commands.
		return 2
	}
}

// loadModel obtains the complete model input from stdin or the injected file
// reader, then delegates byte validation and parsing to lang.Parse.
func loadModel(path string, stdin io.Reader, readFile func(path string) ([]byte, error), stderr io.Writer) (*core.Model, bool) {
	var (
		data []byte
		err  error
	)
	if path == "-" {
		if stdin == nil {
			writeInputDiagnostic(stderr, path, "cannot read the file")
			return nil, false
		}
		data, err = io.ReadAll(stdin)
	} else {
		if readFile == nil {
			writeInputDiagnostic(stderr, path, "cannot read the file")
			return nil, false
		}
		data, err = readFile(path)
	}
	if err != nil {
		writeInputDiagnostic(stderr, path, "cannot read the file")
		return nil, false
	}

	model, modelErr := lang.Parse(string(data))
	if modelErr != nil {
		writeModelDiagnostic(stderr, path, modelErr)
		return nil, false
	}
	return model, true
}

// checkAssignment validates names and values in the command's specified order.
func checkAssignment(model *core.Model, assignments []assignmentText) ([]int, error) {
	valuesByIndex := make(map[int]string, len(assignments))
	seen := make(map[string]struct{}, len(assignments))

	for _, assignment := range assignments {
		index, found := core.VarIndex(model.Vars, assignment.name)
		if !found {
			return nil, core.CliError{Msg: "unknown variable \"" + assignment.name + "\""}
		}
		if _, repeated := seen[assignment.name]; repeated {
			return nil, core.CliError{Msg: "repeated variable \"" + assignment.name + "\""}
		}
		seen[assignment.name] = struct{}{}
		valuesByIndex[index] = assignment.value
	}

	values := make([]int, len(model.Vars))
	for index, variable := range model.Vars {
		valueText, assigned := valuesByIndex[index]
		if !assigned {
			return nil, core.CliError{Msg: "missing value for \"" + variable.Name + "\""}
		}

		minimum := variable.Values[0]
		maximum := variable.Values[len(variable.Values)-1]
		value, validInt := core.ParseInt(valueText, minimum, maximum)
		if !validInt || !containsValue(variable.Values, value) {
			return nil, core.CliError{
				Msg: "value " + valueText + " is not in the domain of \"" + variable.Name + "\" (" + core.DomainText(variable.Values) + ")",
			}
		}
		values[index] = value
	}
	return values, nil
}

// containsValue reports whether an ascending domain contains value.
func containsValue(domain []int, value int) bool {
	for _, candidate := range domain {
		if candidate == value {
			return true
		}
		if candidate > value {
			return false
		}
	}
	return false
}
