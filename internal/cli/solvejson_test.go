package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Gulliby-development/narrowfit/internal/cli"
)

func solveJSONAssert(t *testing.T, args []string, model string, wantCode int, wantStdout, wantStderr string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := cli.Run(args, strings.NewReader(model), &stdout, &stderr, func(string) ([]byte, error) {
		return []byte(model), nil
	})
	if code != wantCode || stdout.String() != wantStdout || stderr.String() != wantStderr {
		t.Fatalf("Run(%q) = (code %d, stdout %q, stderr %q), want (code %d, stdout %q, stderr %q)",
			args, code, stdout.String(), stderr.String(), wantCode, wantStdout, wantStderr)
	}
}

func TestSolveJSONWithoutObjective(t *testing.T) {
	solveJSONAssert(t, []string{"solve", "--json", "m.model"}, "var x 1..2\nvar y 1..2\nrequire x < y\n",
		0, "{\"solution\":{\"x\":1,\"y\":2}}\n", "")
}

func TestSolveJSONWithObjectiveWritesItAfterTheSolution(t *testing.T) {
	solveJSONAssert(t, []string{"solve", "--json", "m.model"}, "var x 1..3\nvar y 1..3\nmaximize x - y\n",
		0, "{\"solution\":{\"x\":3,\"y\":1},\"objective\":2}\n", "")
}

func TestSolveJSONWritesNegativeValuesAndObjective(t *testing.T) {
	solveJSONAssert(t, []string{"solve", "--json", "m.model"}, "var a -3..-1\nminimize a\n",
		0, "{\"solution\":{\"a\":-3},\"objective\":-3}\n", "")
}

func TestSolveJSONKeepsDeclarationOrder(t *testing.T) {
	solveJSONAssert(t, []string{"solve", "--json", "m.model"}, "var b 1\nvar a 2\n",
		0, "{\"solution\":{\"b\":1,\"a\":2}}\n", "")
}

func TestSolveJSONWithoutSolutionWritesNullAndExitsWithOne(t *testing.T) {
	solveJSONAssert(t, []string{"solve", "--json", "m.model"}, "var x 1\nrequire x == 2\n",
		1, "{\"solution\":null}\n", "")
}

func TestSolveJSONMayFollowTheModelAndMayRepeat(t *testing.T) {
	solveJSONAssert(t, []string{"solve", "m.model", "--json"}, "var x 7\n",
		0, "{\"solution\":{\"x\":7}}\n", "")
	solveJSONAssert(t, []string{"solve", "--json", "--json", "m.model"}, "var x 7\n",
		0, "{\"solution\":{\"x\":7}}\n", "")
}

func TestSolveJSONReadsStandardInput(t *testing.T) {
	solveJSONAssert(t, []string{"solve", "--json", "-"}, "var x 4\n",
		0, "{\"solution\":{\"x\":4}}\n", "")
}

func TestSolveWithoutJSONIsUnchanged(t *testing.T) {
	solveJSONAssert(t, []string{"solve", "m.model"}, "var x 1..2\nvar y 1..2\nrequire x < y\n",
		0, "x = 1\ny = 2\n", "")
}

func TestJSONIsAnOptionOfSolveOnly(t *testing.T) {
	solveJSONAssert(t, []string{"all", "--json", "m.model"}, "var x 1\n",
		2, "", "narrowfit: unknown option \"--json\"\n")
	solveJSONAssert(t, []string{"count", "--json", "m.model"}, "var x 1\n",
		2, "", "narrowfit: unknown option \"--json\"\n")
	solveJSONAssert(t, []string{"check", "--json", "m.model", "x=1"}, "var x 1\n",
		2, "", "narrowfit: unknown option \"--json\"\n")
	solveJSONAssert(t, []string{"explain", "--json", "m.model"}, "var x 1\n",
		2, "", "narrowfit: unknown option \"--json\"\n")
	solveJSONAssert(t, []string{"fmt", "--json", "m.model"}, "var x 1\n",
		2, "", "narrowfit: unknown option \"--json\"\n")
}

func TestJSONTakesNoValue(t *testing.T) {
	solveJSONAssert(t, []string{"solve", "--json=1", "m.model"}, "var x 1\n",
		2, "", "narrowfit: unknown option \"--json=1\"\n")
}
