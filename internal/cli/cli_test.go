package cli_test

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/Gulliby-development/narrowfit/internal/cli"
)

var _ func([]string, io.Reader, io.Writer, io.Writer, func(string) ([]byte, error)) int = cli.Run

type cliRunResult struct {
	code      int
	stdout    string
	stderr    string
	fileReads int
	filePath  string
}

func cliRunFile(args []string, data []byte, readErr error) cliRunResult {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	result := cliRunResult{}
	result.code = cli.Run(args, bytes.NewReader(nil), &stdout, &stderr, func(path string) ([]byte, error) {
		result.fileReads++
		result.filePath = path
		return data, readErr
	})
	result.stdout = stdout.String()
	result.stderr = stderr.String()
	return result
}

func cliRunModel(args []string, model string) cliRunResult {
	return cliRunFile(args, []byte(model), nil)
}

func cliRunStdin(args []string, stdin io.Reader) cliRunResult {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	result := cliRunResult{}
	result.code = cli.Run(args, stdin, &stdout, &stderr, func(path string) ([]byte, error) {
		result.fileReads++
		result.filePath = path
		return nil, errors.New("unexpected file read")
	})
	result.stdout = stdout.String()
	result.stderr = stderr.String()
	return result
}

func cliAssertResult(t *testing.T, got cliRunResult, wantCode int, wantStdout, wantStderr string) {
	t.Helper()
	if got.code != wantCode || got.stdout != wantStdout || got.stderr != wantStderr {
		t.Fatalf("Run() = (code %d, stdout %q, stderr %q), want (code %d, stdout %q, stderr %q)",
			got.code, got.stdout, got.stderr, wantCode, wantStdout, wantStderr)
	}
}

func cliAssertFileResult(t *testing.T, got cliRunResult, wantPath string, wantCode int, wantStdout, wantStderr string) {
	t.Helper()
	cliAssertResult(t, got, wantCode, wantStdout, wantStderr)
	if got.fileReads != 1 || got.filePath != wantPath {
		t.Fatalf("readFile calls/path = (%d, %q), want (1, %q)", got.fileReads, got.filePath, wantPath)
	}
}

type cliErrorReader struct{}

func (cliErrorReader) Read([]byte) (int, error) {
	return 0, errors.New("stdin failure")
}

const cliMagicModel = `# 3x3 magic square
var a 1..9
var b 1..9
var c 1..9
var d 1..9
var e 1..9
var f 1..9
var g 1..9
var h 1..9
var i 1..9
alldiff a b c d e f g h i
sum a b c == 15
sum d e f == 15
sum g h i == 15
sum a d g == 15
sum b e h == 15
sum c f i == 15
sum a e i == 15
sum c e g == 15
`

const cliRoomsModel = `# ann, bob and cy each take one of three rooms
var ann 1..3
var bob 1..3
var cy 1..3
alldiff ann bob cy
require ann < bob
require bob < cy
require cy < ann
`

const cliShiftsModel = `# three people on three shifts: 1 early, 2 day, 3 late
var ann 1..3
var bob 1..3
var cy 1..3
var cost 0..100
alldiff ann bob cy
require ann != 3
table cy cost : 1 30 | 2 20 | 3 10
count 1 ann bob cy == 1
minimize cost + bob
`

const cliMessyModel = `var   x   7,5,6,-3..3,10   # overlapping and unordered
var y 0,1,2
var z 1..2
alldiff   x y
sum x  y z >=  -4
count 1 x y == 1
table x y : 1 2 | 3 4
element z in x y is y
require not(x>1 and y<2)or z==2
require x-(y-z)<-(-y)
require (x>1->y>1)->z>0
require x>1->y>1->z>0
require abs(x)<=max(y,2)or z==1
require x*(y+z)%2!=0
minimize -(x+y)*2 / (z % 2 + 1)
`

func TestCLIInputLoadingAndPathDiagnostics(t *testing.T) {
	t.Run("original backslash path is read unchanged and displayed with slashes", func(t *testing.T) {
		got := cliRunFile([]string{"fmt", `folder\model.nf`}, []byte("var x 1..2\n"), nil)
		cliAssertFileResult(t, got, `folder\model.nf`, 0, "var x 1..2\n", "")
	})

	t.Run("file read failure", func(t *testing.T) {
		got := cliRunFile([]string{"solve", `folder\missing.nf`}, nil, errors.New("unavailable"))
		cliAssertFileResult(t, got, `folder\missing.nf`, 2, "", "narrowfit: folder/missing.nf: cannot read the file\n")
	})

	t.Run("stdin read failure uses stdin diagnostic", func(t *testing.T) {
		got := cliRunStdin([]string{"solve", "-"}, cliErrorReader{})
		cliAssertResult(t, got, 2, "", "narrowfit: <stdin>: cannot read the file\n")
		if got.fileReads != 0 {
			t.Fatalf("readFile calls = %d, want none", got.fileReads)
		}
	})

	t.Run("stdin model is read without a file access", func(t *testing.T) {
		got := cliRunStdin([]string{"solve", "-"}, strings.NewReader("var x 1..2\n"))
		cliAssertResult(t, got, 0, "x = 1\n", "")
		if got.fileReads != 0 {
			t.Fatalf("readFile calls = %d, want none", got.fileReads)
		}
	})

	t.Run("malformed stdin bytes are rejected by whole-input parser", func(t *testing.T) {
		stdin := bytes.NewReader([]byte{0x76, 0x61, 0x72, 0x20, 0x78, 0x20, 0x31, 0x2e, 0x2e, 0x32, 0x20, 0x23, 0x20, 0xff, 0x0a})
		got := cliRunStdin([]string{"solve", "-"}, stdin)
		cliAssertResult(t, got, 2, "", "narrowfit: <stdin>: input is not valid UTF-8\n")
		if got.fileReads != 0 {
			t.Fatalf("readFile calls = %d, want none", got.fileReads)
		}
	})

	t.Run("invalid UTF-8 anywhere wins over an earlier bad statement", func(t *testing.T) {
		data := []byte{'b', 'a', 'd', ' ', 's', 't', 'a', 't', 'e', 'm', 'e', 'n', 't', '\n', '#', 0xff, '\n'}
		got := cliRunFile([]string{"fmt", "invalid.nf"}, data, nil)
		cliAssertFileResult(t, got, "invalid.nf", 2, "", "narrowfit: invalid.nf: input is not valid UTF-8\n")
	})

	t.Run("BOM CRLF tabs comments and final line without LF", func(t *testing.T) {
		model := []byte("\xef\xbb\xbfvar\tx\t1..2 # declaration\r\n\trequire x>1")
		got := cliRunFile([]string{"solve", "portable.nf"}, model, nil)
		cliAssertFileResult(t, got, "portable.nf", 0, "x = 2\n", "")
	})

	t.Run("empty comment-only input reports no variables", func(t *testing.T) {
		got := cliRunFile([]string{"fmt", "empty.nf"}, []byte("# no declarations\n\t # comment\n"), nil)
		cliAssertFileResult(t, got, "empty.nf", 2, "", "narrowfit: empty.nf: no variables\n")
	})

	t.Run("requested bad model preserves source line", func(t *testing.T) {
		got := cliRunFile([]string{"solve", "bad.model"}, []byte("var x 1..3\nvar y 3..1"), nil)
		cliAssertFileResult(t, got, "bad.model", 2, "", "narrowfit: bad.model:2: invalid domain \"3..1\"\n")
	})
}

func TestCLISolveSelectionAndObjectives(t *testing.T) {
	cases := []struct {
		name       string
		model      string
		wantCode   int
		wantStdout string
	}{
		{
			name:       "first solution without an objective",
			model:      "var x 1..3\nvar y -1..1\n",
			wantCode:   0,
			wantStdout: "x = 1\ny = -1\n",
		},
		{
			name:       "minimize",
			model:      "var x 1..3\nminimize x\n",
			wantCode:   0,
			wantStdout: "x = 1\nobjective = 1\n",
		},
		{
			name:       "maximize",
			model:      "var x -1..1\nmaximize x\n",
			wantCode:   0,
			wantStdout: "x = 1\nobjective = 1\n",
		},
		{
			name:       "first minimum tie remains selected",
			model:      "var x 1..4\nminimize x % 2\n",
			wantCode:   0,
			wantStdout: "x = 2\nobjective = 0\n",
		},
		{
			name:       "first maximum tie remains selected",
			model:      "var x 1..4\nmaximize -(x % 2)\n",
			wantCode:   0,
			wantStdout: "x = 2\nobjective = 0\n",
		},
		{
			name:       "unevaluable candidates are skipped when another candidate works",
			model:      "var x 0..2\nminimize 10 / x\n",
			wantCode:   0,
			wantStdout: "x = 2\nobjective = 5\n",
		},
		{
			name:       "all objective candidates unevaluable",
			model:      "var x 1..2\nminimize 1 / 0\n",
			wantCode:   1,
			wantStdout: "no solution\n",
		},
		{
			name:       "unsatisfiable model",
			model:      "var x 1..2\nrequire x > 2\n",
			wantCode:   1,
			wantStdout: "no solution\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := cliRunModel([]string{"solve", "model.nf"}, tc.model)
			cliAssertFileResult(t, got, "model.nf", tc.wantCode, tc.wantStdout, "")
		})
	}

	t.Run("requested shifts example", func(t *testing.T) {
		got := cliRunModel([]string{"solve", "shifts.model"}, cliShiftsModel)
		cliAssertFileResult(t, got, "shifts.model", 0,
			"ann = 2\nbob = 1\ncy = 3\ncost = 10\nobjective = 11\n", "")
	})
}

func TestCLIAllAndCountBoundaries(t *testing.T) {
	cases := []struct {
		name       string
		args       []string
		model      string
		wantCode   int
		wantStdout string
	}{
		{
			name:       "all has zero solutions",
			args:       []string{"all", "--limit", "2", "model.nf"},
			model:      "var x 1..2\nrequire x > 2\n",
			wantCode:   1,
			wantStdout: "0 solutions\n",
		},
		{
			name:       "count has zero solutions",
			args:       []string{"count", "--limit=2", "model.nf"},
			model:      "var x 1..2\nrequire x > 2\n",
			wantCode:   1,
			wantStdout: "0 solutions\n",
		},
		{
			name:       "all has one solution and singular summary",
			args:       []string{"all", "--limit=1", "model.nf"},
			model:      "var x 1..2\nrequire x == 2\n",
			wantCode:   0,
			wantStdout: "1: x=2\n1 solution\n",
		},
		{
			name:       "count has one solution and singular summary",
			args:       []string{"count", "model.nf"},
			model:      "var x 1..2\nrequire x == 2\n",
			wantCode:   0,
			wantStdout: "1 solution\n",
		},
		{
			name:       "all exactly at limit does not claim more",
			args:       []string{"all", "--limit=2", "model.nf"},
			model:      "var x 1..2\n",
			wantCode:   0,
			wantStdout: "1: x=1\n2: x=2\n2 solutions\n",
		},
		{
			name:       "count exactly at limit does not claim more",
			args:       []string{"count", "model.nf", "--limit", "2"},
			model:      "var x 1..2\n",
			wantCode:   0,
			wantStdout: "2 solutions\n",
		},
		{
			name:       "all probes the solution after the limit",
			args:       []string{"all", "--limit", "2", "model.nf"},
			model:      "var x 1..3\n",
			wantCode:   0,
			wantStdout: "1: x=1\n2: x=2\nmore than 2 solutions\n",
		},
		{
			name:       "count probes the solution after the limit",
			args:       []string{"count", "--limit=2", "model.nf"},
			model:      "var x 1..3\n",
			wantCode:   0,
			wantStdout: "more than 2 solutions\n",
		},
		{
			name:       "all ignores objective and retains declaration order",
			args:       []string{"all", "--limit=1", "model.nf"},
			model:      "var x 1..3\nmaximize x\n",
			wantCode:   0,
			wantStdout: "1: x=1\nmore than 1 solutions\n",
		},
		{
			name:       "count ignores objective and counts every assignment",
			args:       []string{"count", "--limit=3", "model.nf"},
			model:      "var x 1..3\nmaximize x\n",
			wantCode:   0,
			wantStdout: "3 solutions\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := cliRunModel(tc.args, tc.model)
			cliAssertFileResult(t, got, "model.nf", tc.wantCode, tc.wantStdout, "")
		})
	}

	t.Run("requested magic count example", func(t *testing.T) {
		got := cliRunModel([]string{"count", "magic.model"}, cliMagicModel)
		cliAssertFileResult(t, got, "magic.model", 0, "8 solutions\n", "")
	})

	t.Run("requested magic all limit two example", func(t *testing.T) {
		got := cliRunModel([]string{"all", "--limit", "2", "magic.model"}, cliMagicModel)
		want := "1: a=2 b=7 c=6 d=9 e=5 f=1 g=4 h=3 i=8\n" +
			"2: a=2 b=9 c=4 d=7 e=5 f=3 g=6 h=1 i=8\n" +
			"more than 2 solutions\n"
		cliAssertFileResult(t, got, "magic.model", 0, want, "")
	})

	t.Run("worked magic solve output", func(t *testing.T) {
		got := cliRunModel([]string{"solve", "magic.model"}, cliMagicModel)
		want := "a = 2\nb = 7\nc = 6\nd = 9\ne = 5\nf = 1\ng = 4\nh = 3\ni = 8\n"
		cliAssertFileResult(t, got, "magic.model", 0, want, "")
	})

	t.Run("requested shifts count example", func(t *testing.T) {
		got := cliRunModel([]string{"count", "shifts.model"}, cliShiftsModel)
		cliAssertFileResult(t, got, "shifts.model", 0, "4 solutions\n", "")
	})
}

func TestCLICheckAssignmentsAndWorkedExample(t *testing.T) {
	t.Run("requested magic satisfying assignment", func(t *testing.T) {
		got := cliRunModel([]string{"check", "magic.model", "a=2", "b=7", "c=6", "d=9", "e=5", "f=1", "g=4", "h=3", "i=8"}, cliMagicModel)
		cliAssertFileResult(t, got, "magic.model", 0, "ok\n", "")
	})

	t.Run("requested magic violating assignment prints every violation", func(t *testing.T) {
		got := cliRunModel([]string{"check", "magic.model", "a=2", "b=7", "c=6", "d=9", "e=5", "f=1", "g=4", "h=3", "i=9"}, cliMagicModel)
		want := "violated 11: alldiff a b c d e f g h i\n" +
			"violated 14: sum g h i == 15\n" +
			"violated 17: sum c f i == 15\n" +
			"violated 18: sum a e i == 15\n"
		cliAssertFileResult(t, got, "magic.model", 1, want, "")
	})

	t.Run("division by zero makes a require unsatisfied", func(t *testing.T) {
		model := "var x 1..3\nrequire x / 0 == 1\n"
		got := cliRunModel([]string{"solve", "m.model"}, model)
		cliAssertFileResult(t, got, "m.model", 1, "no solution\n", "")
	})
}

func TestCLIExplainAndFmt(t *testing.T) {
	t.Run("requested rooms solve reports no solution", func(t *testing.T) {
		got := cliRunModel([]string{"solve", "rooms.model"}, cliRoomsModel)
		cliAssertFileResult(t, got, "rooms.model", 1, "no solution\n", "")
	})

	t.Run("requested rooms conflict preserves exact statement lines", func(t *testing.T) {
		got := cliRunModel([]string{"explain", "rooms.model"}, cliRoomsModel)
		want := "conflict of 3 constraints\n" +
			"6: require ann < bob\n" +
			"7: require bob < cy\n" +
			"8: require cy < ann\n"
		cliAssertFileResult(t, got, "rooms.model", 1, want, "")
	})

	t.Run("satisfiable explanation ignores objective", func(t *testing.T) {
		model := "var x 1..2\nrequire 1 < 2\nmaximize x\n"
		got := cliRunModel([]string{"explain", "model.nf"}, model)
		cliAssertFileResult(t, got, "model.nf", 0, "satisfiable\n", "")
	})

	t.Run("single-constraint conflict uses singular wording", func(t *testing.T) {
		model := "var x 1..1\nrequire x == 2\n"
		got := cliRunModel([]string{"explain", "model.nf"}, model)
		cliAssertFileResult(t, got, "model.nf", 1, "conflict of 1 constraint\n2: require x == 2\n", "")
	})

	t.Run("multiple conflicts preserve model statement order and ignore objective", func(t *testing.T) {
		model := "var x 1..2\nrequire x == 1\nrequire x == 2\nminimize x\n"
		got := cliRunModel([]string{"explain", "model.nf"}, model)
		want := "conflict of 2 constraints\n2: require x == 1\n3: require x == 2\n"
		cliAssertFileResult(t, got, "model.nf", 1, want, "")
	})

	t.Run("empty-scope false requirement remains a conflict", func(t *testing.T) {
		model := "var x 1..2\nrequire 1 < 0\n"
		got := cliRunModel([]string{"explain", "model.nf"}, model)
		cliAssertFileResult(t, got, "model.nf", 1, "conflict of 1 constraint\n2: require 1 < 0\n", "")
	})

	t.Run("requested messy model formats to exact canonical text", func(t *testing.T) {
		got := cliRunModel([]string{"fmt", "messy.model"}, cliMessyModel)
		want := "var x -3..3,5..7,10\n" +
			"var y 0..2\n" +
			"var z 1..2\n" +
			"alldiff x y\n" +
			"sum x y z >= -4\n" +
			"count 1 x y == 1\n" +
			"table x y : 1 2 | 3 4\n" +
			"element z in x y is y\n" +
			"require not (x > 1 and y < 2) or z == 2\n" +
			"require x - (y - z) < -(-y)\n" +
			"require (x > 1 -> y > 1) -> z > 0\n" +
			"require x > 1 -> y > 1 -> z > 0\n" +
			"require abs(x) <= max(y, 2) or z == 1\n" +
			"require x * (y + z) % 2 != 0\n" +
			"minimize -(x + y) * 2 / (z % 2 + 1)\n"
		cliAssertFileResult(t, got, "messy.model", 0, want, "")
	})

	t.Run("fmt prints unsatisfiable model without solving", func(t *testing.T) {
		model := "var x 1..1\nrequire x == 2\n"
		got := cliRunModel([]string{"fmt", "unsat.nf"}, model)
		cliAssertFileResult(t, got, "unsat.nf", 0, "var x 1\nrequire x == 2\n", "")
	})
}
