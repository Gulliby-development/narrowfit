package cli

import (
	"strings"

	"github.com/Gulliby-development/narrowfit/internal/core"
)

const (
	defaultAllLimit   = 20
	defaultCountLimit = 1000000
	maximumLimit      = 1000000
)

// assignmentText retains the spelling of an assignment value until the model
// domain is known. In particular, the value can be wider than a machine int.
type assignmentText struct {
	argument string
	name     string
	value    string
}

// cliArgs contains arguments after command syntax and arity have been checked.
type cliArgs struct {
	command     string
	modelPath   string
	limit       int
	fmtCheck    bool
	jsonOutput  bool
	checkTotal  bool
	assignments []assignmentText
}

// parseArgs validates command-line arguments without performing input access.
// The help result is true only for standalone --help.
func parseArgs(args []string) (parsed cliArgs, help bool, err error) {
	if len(args) == 0 {
		return parsed, false, core.CliError{Msg: "missing command (solve, all, count, check, explain or fmt)"}
	}
	if len(args) == 1 && args[0] == "--help" {
		return parsed, true, nil
	}

	command := args[0]
	switch command {
	case "solve", "all", "count", "check", "explain", "fmt":
	default:
		return parsed, false, core.CliError{Msg: "unknown command \"" + command + "\""}
	}

	parsed.command = command
	switch command {
	case "all":
		parsed.limit = defaultAllLimit
	case "count":
		parsed.limit = defaultCountLimit
	}

	positionals := make([]string, 0, len(args)-1)
	hasLimit := false
	var limitText string
	limitSupported := command == "all" || command == "count"

	for i := 1; i < len(args); i++ {
		arg := args[i]
		if command == "fmt" && arg == "--check" {
			parsed.fmtCheck = true
			continue
		}
		if command == "solve" && arg == "--json" {
			parsed.jsonOutput = true
			continue
		}
		if command == "check" && arg == "--total" {
			parsed.checkTotal = true
			continue
		}
		if limitSupported && arg == "--limit" {
			if i+1 == len(args) {
				return cliArgs{}, false, core.CliError{Msg: "--limit needs a value"}
			}
			i++
			limitText = args[i]
			hasLimit = true
			continue
		}
		if limitSupported && strings.HasPrefix(arg, "--limit=") {
			limitText = strings.TrimPrefix(arg, "--limit=")
			hasLimit = true
			continue
		}
		if strings.HasPrefix(arg, "-") && arg != "-" {
			return cliArgs{}, false, core.CliError{Msg: "unknown option \"" + arg + "\""}
		}
		positionals = append(positionals, arg)
	}

	if hasLimit {
		limit, ok := core.ParseInt(limitText, 1, maximumLimit)
		if !ok {
			return cliArgs{}, false, core.CliError{Msg: "invalid --limit \"" + limitText + "\""}
		}
		parsed.limit = limit
	}

	if len(positionals) == 0 {
		return cliArgs{}, false, core.CliError{Msg: "missing MODEL"}
	}
	parsed.modelPath = positionals[0]

	if command != "check" && len(positionals) > 1 {
		return cliArgs{}, false, core.CliError{Msg: "too many arguments"}
	}
	if command == "check" {
		for _, arg := range positionals[1:] {
			assignment, ok := parseAssignmentText(arg)
			if !ok {
				return cliArgs{}, false, core.CliError{Msg: "invalid assignment \"" + arg + "\""}
			}
			parsed.assignments = append(parsed.assignments, assignment)
		}
	}

	return parsed, false, nil
}

// parseAssignmentText checks the assignment grammar without converting the
// integer. A value is a canonical decimal integer of any magnitude.
func parseAssignmentText(argument string) (assignmentText, bool) {
	if strings.Count(argument, "=") != 1 {
		return assignmentText{}, false
	}
	separator := strings.IndexByte(argument, '=')
	if separator == 0 || !canonicalIntegerText(argument[separator+1:]) {
		return assignmentText{}, false
	}
	return assignmentText{
		argument: argument,
		name:     argument[:separator],
		value:    argument[separator+1:],
	}, true
}

func canonicalIntegerText(text string) bool {
	if text == "" {
		return false
	}

	start := 0
	negative := text[0] == '-'
	if negative {
		start = 1
		if start == len(text) {
			return false
		}
	}
	if text[start] == '0' {
		return !negative && start+1 == len(text)
	}
	if text[start] < '1' || text[start] > '9' {
		return false
	}
	for i := start + 1; i < len(text); i++ {
		if text[i] < '0' || text[i] > '9' {
			return false
		}
	}
	return true
}
