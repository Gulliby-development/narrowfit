package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Gulliby-development/narrowfit/internal/cli"
)

func checkTotalRow(t *testing.T, args []string, model string, wantCode int, wantStdout, wantStderr string) {
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

func TestCheckTotalRows(t *testing.T) {
	checkTotalRow(t, []string{"check", "--total", "m.model", "a=1", "b=2", "c=3"},
		"var a 1..3\nvar b 1..3\nvar c 1..3\nalldiff a b c\nsum a b c == 6\nrequire a < c\n", 0, "0 of 3 violated\n", "")
	checkTotalRow(t, []string{"check", "--total", "m.model", "a=1", "b=1", "c=1"},
		"var a 1..3\nvar b 1..3\nvar c 1..3\nalldiff a b c\nsum a b c == 6\nrequire a < c\n", 1, "3 of 3 violated\n", "")
	checkTotalRow(t, []string{"check", "--total", "m.model", "a=3", "b=2", "c=1"},
		"var a 1..3\nvar b 1..3\nvar c 1..3\nalldiff a b c\nsum a b c == 6\nrequire a < c\n", 1, "1 of 3 violated\n", "")
	checkTotalRow(t, []string{"check", "m.model", "a=2", "b=2", "c=2", "--total"},
		"var a 1..3\nvar b 1..3\nvar c 1..3\nalldiff a b c\nsum a b c == 6\nrequire a < c\n", 1, "2 of 3 violated\n", "")
	checkTotalRow(t, []string{"check", "--total", "--total", "m.model", "a=1", "b=3", "c=2"},
		"var a 1..3\nvar b 1..3\nvar c 1..3\nalldiff a b c\nsum a b c == 6\nrequire a < c\n", 0, "0 of 3 violated\n", "")
	checkTotalRow(t, []string{"check", "m.model", "--total", "a=2", "b=1", "c=3"},
		"var a 1..3\nvar b 1..3\nvar c 1..3\nalldiff a b c\nsum a b c == 6\nrequire a < c\n", 0, "0 of 3 violated\n", "")
	checkTotalRow(t, []string{"check", "--total", "m.model", "x=2"},
		"var x 1..3\n", 0, "0 of 0 violated\n", "")
	checkTotalRow(t, []string{"check", "--total", "m.model", "x=2", "y=2"},
		"var x 1..3\nvar y 1..3\nrequire x != y\nmaximize x\n", 1, "1 of 1 violated\n", "")
	checkTotalRow(t, []string{"check", "--total", "m.model", "a=2", "b=7", "c=6", "d=9", "e=5", "f=1", "g=4", "h=3", "i=8"},
		"# 3x3 magic square\nvar a 1..9\nvar b 1..9\nvar c 1..9\nvar d 1..9\nvar e 1..9\nvar f 1..9\nvar g 1..9\nvar h 1..9\nvar i 1..9\nalldiff a b c d e f g h i\nsum a b c == 15\nsum d e f == 15\nsum g h i == 15\nsum a d g == 15\nsum b e h == 15\nsum c f i == 15\nsum a e i == 15\nsum c e g == 15\n", 0, "0 of 9 violated\n", "")
	checkTotalRow(t, []string{"check", "--total", "m.model", "a=1", "b=2", "c=3", "d=4", "e=5", "f=6", "g=7", "h=8", "i=9"},
		"# 3x3 magic square\nvar a 1..9\nvar b 1..9\nvar c 1..9\nvar d 1..9\nvar e 1..9\nvar f 1..9\nvar g 1..9\nvar h 1..9\nvar i 1..9\nalldiff a b c d e f g h i\nsum a b c == 15\nsum d e f == 15\nsum g h i == 15\nsum a d g == 15\nsum b e h == 15\nsum c f i == 15\nsum a e i == 15\nsum c e g == 15\n", 1, "4 of 9 violated\n", "")
	checkTotalRow(t, []string{"check", "--total", "m.model", "z=1", "a=1", "b=2", "c=3"},
		"var a 1..3\nvar b 1..3\nvar c 1..3\nalldiff a b c\nsum a b c == 6\nrequire a < c\n", 2, "", "narrowfit: unknown variable \"z\"\n")
	checkTotalRow(t, []string{"check", "--total", "m.model", "a=1", "b=2"},
		"var a 1..3\nvar b 1..3\nvar c 1..3\nalldiff a b c\nsum a b c == 6\nrequire a < c\n", 2, "", "narrowfit: missing value for \"c\"\n")
	checkTotalRow(t, []string{"check", "--total", "m.model", "a=7", "b=2", "c=3"},
		"var a 1..3\nvar b 1..3\nvar c 1..3\nalldiff a b c\nsum a b c == 6\nrequire a < c\n", 2, "", "narrowfit: value 7 is not in the domain of \"a\" (1..3)\n")
	checkTotalRow(t, []string{"check", "--total", "m.model", "a"},
		"var a 1..3\nvar b 1..3\nvar c 1..3\nalldiff a b c\nsum a b c == 6\nrequire a < c\n", 2, "", "narrowfit: invalid assignment \"a\"\n")
	checkTotalRow(t, []string{"check", "--total"},
		"var a 1..3\nvar b 1..3\nvar c 1..3\nalldiff a b c\nsum a b c == 6\nrequire a < c\n", 2, "", "narrowfit: missing MODEL\n")
	checkTotalRow(t, []string{"check", "--total=1", "m.model", "a=1", "b=2", "c=3"},
		"var a 1..3\nvar b 1..3\nvar c 1..3\nalldiff a b c\nsum a b c == 6\nrequire a < c\n", 2, "", "narrowfit: unknown option \"--total=1\"\n")
	checkTotalRow(t, []string{"solve", "--total", "m.model"},
		"var a 1..3\nvar b 1..3\nvar c 1..3\nalldiff a b c\nsum a b c == 6\nrequire a < c\n", 2, "", "narrowfit: unknown option \"--total\"\n")
	checkTotalRow(t, []string{"fmt", "--total", "m.model"},
		"var a 1..3\nvar b 1..3\nvar c 1..3\nalldiff a b c\nsum a b c == 6\nrequire a < c\n", 2, "", "narrowfit: unknown option \"--total\"\n")
	checkTotalRow(t, []string{"all", "--total", "m.model"},
		"var a 1..3\nvar b 1..3\nvar c 1..3\nalldiff a b c\nsum a b c == 6\nrequire a < c\n", 2, "", "narrowfit: unknown option \"--total\"\n")
	checkTotalRow(t, []string{"check", "--totals", "m.model", "a=1", "b=2", "c=3"},
		"var a 1..3\nvar b 1..3\nvar c 1..3\nalldiff a b c\nsum a b c == 6\nrequire a < c\n", 2, "", "narrowfit: unknown option \"--totals\"\n")
	checkTotalRow(t, []string{"check", "--total", "-", "x=3"},
		"var x 1..3\n", 0, "0 of 0 violated\n", "")
}
