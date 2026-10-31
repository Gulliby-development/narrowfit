package cli_test

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/Gulliby-development/narrowfit/internal/cli"
)

const (
	argsDefaultModel = "var x 1..2\n"
	argsCheckModel   = "var x 1..9\nvar y 1..9\nrequire x < y\n"
	argsUsage        = "usage: narrowfit solve [--json] MODEL\n" +
		"       narrowfit all [--limit N] MODEL\n" +
		"       narrowfit count [--limit N] MODEL\n" +
		"       narrowfit check [--total] MODEL NAME=VALUE...\n" +
		"       narrowfit explain MODEL\n" +
		"       narrowfit fmt [--check] MODEL\n"
)

type argsTrackingReader struct {
	reader *strings.Reader
	reads  int
}

func (r *argsTrackingReader) Read(p []byte) (int, error) {
	r.reads++
	return r.reader.Read(p)
}

type argsRunResult struct {
	code       int
	stdout     string
	stderr     string
	fileReads  int
	filePath   string
	stdinReads int
}

type argsContractCase struct {
	name          string
	args          []string
	model         string
	wantCode      int
	wantStdout    string
	wantStderr    string
	wantFilePath  string
	wantStdinRead bool
}

func argsRun(args []string, model string) argsRunResult {
	if model == "" {
		model = argsDefaultModel
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	stdin := &argsTrackingReader{reader: strings.NewReader(model)}
	result := argsRunResult{}
	result.code = cli.Run(args, stdin, &stdout, &stderr, func(path string) ([]byte, error) {
		result.fileReads++
		result.filePath = path
		return []byte(model), nil
	})
	result.stdout = stdout.String()
	result.stderr = stderr.String()
	result.stdinReads = stdin.reads
	return result
}

func argsAssertCase(t *testing.T, tc argsContractCase) {
	t.Helper()
	got := argsRun(tc.args, tc.model)
	if got.code != tc.wantCode || got.stdout != tc.wantStdout || got.stderr != tc.wantStderr {
		t.Fatalf("Run(%q) = (code %d, stdout %q, stderr %q), want (code %d, stdout %q, stderr %q)",
			tc.args, got.code, got.stdout, got.stderr, tc.wantCode, tc.wantStdout, tc.wantStderr)
	}
	if tc.wantFilePath == "" {
		if got.fileReads != 0 {
			t.Fatalf("readFile calls = %d, want none", got.fileReads)
		}
	} else if got.fileReads != 1 || got.filePath != tc.wantFilePath {
		t.Fatalf("readFile calls/path = (%d, %q), want (1, %q)", got.fileReads, got.filePath, tc.wantFilePath)
	}
	if tc.wantStdinRead {
		if got.stdinReads == 0 {
			t.Fatal("stdin was not read")
		}
	} else if got.stdinReads != 0 {
		t.Fatalf("stdin reads = %d, want none", got.stdinReads)
	}
}

func TestArgsHelpAndCommandPrecedence(t *testing.T) {
	cases := []argsContractCase{
		{
			name:       "standalone help",
			args:       []string{"--help"},
			wantCode:   0,
			wantStdout: argsUsage,
		},
		{
			name:       "missing command",
			args:       []string{},
			wantCode:   2,
			wantStderr: "narrowfit: missing command (solve, all, count, check, explain or fmt)\n",
		},
		{
			name:       "unknown command",
			args:       []string{"launch"},
			wantCode:   2,
			wantStderr: "narrowfit: unknown command \"launch\"\n",
		},
		{
			name:       "unknown command placeholder argument",
			args:       []string{"ARG"},
			wantCode:   2,
			wantStderr: "narrowfit: unknown command \"ARG\"\n",
		},
		{
			name:       "help is not standalone when followed by an argument",
			args:       []string{"--help", "m.model"},
			wantCode:   2,
			wantStderr: "narrowfit: unknown command \"--help\"\n",
		},
		{
			name:       "unknown command wins before an option error",
			args:       []string{"unknown", "--limit"},
			wantCode:   2,
			wantStderr: "narrowfit: unknown command \"unknown\"\n",
		},
		{
			name:       "help among command arguments is an unknown option",
			args:       []string{"solve", "m.model", "--help"},
			wantCode:   2,
			wantStderr: "narrowfit: unknown option \"--help\"\n",
		},
		{
			name:       "short option is reported as typed",
			args:       []string{"solve", "-x", "m.model"},
			wantCode:   2,
			wantStderr: "narrowfit: unknown option \"-x\"\n",
		},
		{
			name:       "limit option is unsupported for solve",
			args:       []string{"solve", "--limit=2", "m.model"},
			wantCode:   2,
			wantStderr: "narrowfit: unknown option \"--limit=2\"\n",
		},
		{
			name:          "minus is a model path rather than an option",
			args:          []string{"count", "-"},
			model:         "var x 7\n",
			wantCode:      0,
			wantStdout:    "1 solution\n",
			wantStdinRead: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			argsAssertCase(t, tc)
		})
	}
}

func TestArgsLimitOptionsAndDiagnostics(t *testing.T) {
	manySolutions := "var x 1..21\n"
	cases := []argsContractCase{
		{
			name:         "separate limit before model",
			args:         []string{"all", "--limit", "1", "m.model"},
			wantCode:     0,
			wantStdout:   "1: x=1\nmore than 1 solutions\n",
			wantFilePath: "m.model",
		},
		{
			name:         "inline limit after model",
			args:         []string{"count", "m.model", "--limit=1"},
			wantCode:     0,
			wantStdout:   "more than 1 solutions\n",
			wantFilePath: "m.model",
		},
		{
			name:         "maximum limit is accepted inline",
			args:         []string{"all", "--limit=1000000", "m.model"},
			wantCode:     0,
			wantStdout:   "1: x=1\n2: x=2\n2 solutions\n",
			wantFilePath: "m.model",
		},
		{
			name:         "maximum limit is accepted as a separate value",
			args:         []string{"count", "m.model", "--limit", "1000000"},
			wantCode:     0,
			wantStdout:   "2 solutions\n",
			wantFilePath: "m.model",
		},
		{
			name:         "last limit wins when the first is invalid",
			args:         []string{"count", "--limit", "0", "m.model", "--limit=2"},
			wantCode:     0,
			wantStdout:   "2 solutions\n",
			wantFilePath: "m.model",
		},
		{
			name:       "last invalid limit is checked",
			args:       []string{"all", "--limit=2", "m.model", "--limit", "x"},
			wantCode:   2,
			wantStderr: "narrowfit: invalid --limit \"x\"\n",
		},
		{
			name:       "zero limit",
			args:       []string{"count", "m.model", "--limit", "0"},
			wantCode:   2,
			wantStderr: "narrowfit: invalid --limit \"0\"\n",
		},
		{
			name:       "negative limit is consumed as the option value",
			args:       []string{"count", "--limit", "-1", "m.model"},
			wantCode:   2,
			wantStderr: "narrowfit: invalid --limit \"-1\"\n",
		},
		{
			name:       "limit above maximum",
			args:       []string{"all", "m.model", "--limit=1000001"},
			wantCode:   2,
			wantStderr: "narrowfit: invalid --limit \"1000001\"\n",
		},
		{
			name:       "plus sign is not a canonical limit",
			args:       []string{"all", "--limit=+3", "m.model"},
			wantCode:   2,
			wantStderr: "narrowfit: invalid --limit \"+3\"\n",
		},
		{
			name:       "leading zero is not a canonical limit",
			args:       []string{"count", "--limit", "007", "m.model"},
			wantCode:   2,
			wantStderr: "narrowfit: invalid --limit \"007\"\n",
		},
		{
			name:       "negative zero is not a canonical limit",
			args:       []string{"all", "--limit=-0", "m.model"},
			wantCode:   2,
			wantStderr: "narrowfit: invalid --limit \"-0\"\n",
		},
		{
			name:       "empty inline limit",
			args:       []string{"all", "m.model", "--limit="},
			wantCode:   2,
			wantStderr: "narrowfit: invalid --limit \"\"\n",
		},
		{
			name:       "non-numeric limit",
			args:       []string{"count", "--limit=x", "m.model"},
			wantCode:   2,
			wantStderr: "narrowfit: invalid --limit \"x\"\n",
		},
		{
			name:       "option value beginning with hyphen is validated as a value",
			args:       []string{"all", "m.model", "--limit", "-x"},
			wantCode:   2,
			wantStderr: "narrowfit: invalid --limit \"-x\"\n",
		},
		{
			name:       "missing separate limit value",
			args:       []string{"all", "m.model", "--limit"},
			wantCode:   2,
			wantStderr: "narrowfit: --limit needs a value\n",
		},
		{
			name:       "earlier unknown option wins over trailing missing limit",
			args:       []string{"count", "-x", "m.model", "--limit"},
			wantCode:   2,
			wantStderr: "narrowfit: unknown option \"-x\"\n",
		},
		{
			name:       "invalid final limit wins before missing model",
			args:       []string{"all", "--limit", "0"},
			wantCode:   2,
			wantStderr: "narrowfit: invalid --limit \"0\"\n",
		},
		{
			name:       "missing model follows option validation",
			args:       []string{"count", "--limit", "1"},
			wantCode:   2,
			wantStderr: "narrowfit: missing MODEL\n",
		},
		{
			name:       "missing model",
			args:       []string{"solve"},
			wantCode:   2,
			wantStderr: "narrowfit: missing MODEL\n",
		},
		{
			name:       "too many positionals",
			args:       []string{"solve", "m.model", "extra"},
			wantCode:   2,
			wantStderr: "narrowfit: too many arguments\n",
		},
		{
			name:       "too many positionals wins before any model read",
			args:       []string{"fmt", "m.model", "extra", "another"},
			wantCode:   2,
			wantStderr: "narrowfit: too many arguments\n",
		},
		{
			name:         "default all limit truncates at twenty",
			args:         []string{"all", "many.model"},
			model:        manySolutions,
			wantCode:     0,
			wantStdout:   "1: x=1\n2: x=2\n3: x=3\n4: x=4\n5: x=5\n6: x=6\n7: x=7\n8: x=8\n9: x=9\n10: x=10\n11: x=11\n12: x=12\n13: x=13\n14: x=14\n15: x=15\n16: x=16\n17: x=17\n18: x=18\n19: x=19\n20: x=20\nmore than 20 solutions\n",
			wantFilePath: "many.model",
		},
		{
			name:         "default count limit allows all twenty-one solutions",
			args:         []string{"count", "many.model"},
			model:        manySolutions,
			wantCode:     0,
			wantStdout:   "21 solutions\n",
			wantFilePath: "many.model",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			argsAssertCase(t, tc)
		})
	}
}

func TestArgsAssignmentSyntaxValidation(t *testing.T) {
	cases := []argsContractCase{
		{
			name:       "malformed assignment with an empty value",
			args:       []string{"check", "m.model", "x="},
			wantCode:   2,
			wantStderr: "narrowfit: invalid assignment \"x=\"\n",
		},
		{
			name:       "malformed assignment with an empty name",
			args:       []string{"check", "m.model", "=1"},
			wantCode:   2,
			wantStderr: "narrowfit: invalid assignment \"=1\"\n",
		},
		{
			name:       "malformed assignment with multiple separators",
			args:       []string{"check", "m.model", "x=1=2"},
			wantCode:   2,
			wantStderr: "narrowfit: invalid assignment \"x=1=2\"\n",
		},
		{
			name:       "plus-prefixed assignment value is malformed",
			args:       []string{"check", "m.model", "x=+3"},
			wantCode:   2,
			wantStderr: "narrowfit: invalid assignment \"x=+3\"\n",
		},
		{
			name:       "leading-zero assignment value is malformed",
			args:       []string{"check", "m.model", "x=007"},
			wantCode:   2,
			wantStderr: "narrowfit: invalid assignment \"x=007\"\n",
		},
		{
			name:       "negative-zero assignment value is malformed",
			args:       []string{"check", "m.model", "x=-0"},
			wantCode:   2,
			wantStderr: "narrowfit: invalid assignment \"x=-0\"\n",
		},
		{
			name:       "non-digit assignment value is malformed",
			args:       []string{"check", "m.model", "x=3x"},
			wantCode:   2,
			wantStderr: "narrowfit: invalid assignment \"x=3x\"\n",
		},
		{
			name:       "invalid assignment precedes any input access",
			args:       []string{"check", "missing.model", "x=+"},
			wantCode:   2,
			wantStderr: "narrowfit: invalid assignment \"x=+\"\n",
		},
		{
			name:       "first malformed assignment wins",
			args:       []string{"check", "m.model", "x=1", "y=007", "z=+"},
			wantCode:   2,
			wantStderr: "narrowfit: invalid assignment \"y=007\"\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			argsAssertCase(t, tc)
		})
	}
}

func TestArgsCheckAssignmentOrderAndValues(t *testing.T) {
	cases := []argsContractCase{
		{
			name:         "canonical zero is accepted",
			args:         []string{"check", "m.model", "x=0"},
			model:        "var x -3..0\n",
			wantCode:     0,
			wantStdout:   "ok\n",
			wantFilePath: "m.model",
		},
		{
			name:         "canonical negative integer is accepted",
			args:         []string{"check", "m.model", "x=-3"},
			model:        "var x -3..0\n",
			wantCode:     0,
			wantStdout:   "ok\n",
			wantFilePath: "m.model",
		},
		{
			name:         "unknown variable is reported",
			args:         []string{"check", "m.model", "q=1", "x=1", "y=2"},
			model:        argsCheckModel,
			wantCode:     2,
			wantStderr:   "narrowfit: unknown variable \"q\"\n",
			wantFilePath: "m.model",
		},
		{
			name:         "repetition wins before a later unknown variable",
			args:         []string{"check", "m.model", "x=1", "x=2", "q=3"},
			model:        argsCheckModel,
			wantCode:     2,
			wantStderr:   "narrowfit: repeated variable \"x\"\n",
			wantFilePath: "m.model",
		},
		{
			name:         "unknown variable wins before a later repetition",
			args:         []string{"check", "m.model", "x=1", "q=2", "x=3"},
			model:        argsCheckModel,
			wantCode:     2,
			wantStderr:   "narrowfit: unknown variable \"q\"\n",
			wantFilePath: "m.model",
		},
		{
			name:         "unknown names are checked before a missing declaration value",
			args:         []string{"check", "m.model", "q=1"},
			model:        argsCheckModel,
			wantCode:     2,
			wantStderr:   "narrowfit: unknown variable \"q\"\n",
			wantFilePath: "m.model",
		},
		{
			name:         "missing earlier variable wins over a later domain error",
			args:         []string{"check", "m.model", "y=11"},
			model:        argsCheckModel,
			wantCode:     2,
			wantStderr:   "narrowfit: missing value for \"x\"\n",
			wantFilePath: "m.model",
		},
		{
			name:         "earlier domain error wins over a later missing variable",
			args:         []string{"check", "m.model", "x=11"},
			model:        argsCheckModel,
			wantCode:     2,
			wantStderr:   "narrowfit: value 11 is not in the domain of \"x\" (1..9)\n",
			wantFilePath: "m.model",
		},
		{
			name:         "declaration order wins over assignment order",
			args:         []string{"check", "m.model", "y=11", "x=11"},
			model:        argsCheckModel,
			wantCode:     2,
			wantStderr:   "narrowfit: value 11 is not in the domain of \"x\" (1..9)\n",
			wantFilePath: "m.model",
		},
		{
			name:         "second declared variable is required",
			args:         []string{"check", "m.model", "x=1"},
			model:        argsCheckModel,
			wantCode:     2,
			wantStderr:   "narrowfit: missing value for \"y\"\n",
			wantFilePath: "m.model",
		},
		{
			name:         "canonical disjoint domain appears in diagnostic",
			args:         []string{"check", "m.model", "x=4"},
			model:        "var x 7,5,6,1..3\n",
			wantCode:     2,
			wantStderr:   "narrowfit: value 4 is not in the domain of \"x\" (1..3,5..7)\n",
			wantFilePath: "m.model",
		},
		{
			name:         "large positive canonical integer retains its full spelling",
			args:         []string{"check", "m.model", "x=9223372036854775808"},
			model:        "var x 1..9\n",
			wantCode:     2,
			wantStderr:   "narrowfit: value 9223372036854775808 is not in the domain of \"x\" (1..9)\n",
			wantFilePath: "m.model",
		},
		{
			name:         "large negative canonical integer retains its full spelling",
			args:         []string{"check", "m.model", "x=-9223372036854775809"},
			model:        "var x -9..-1\n",
			wantCode:     2,
			wantStderr:   "narrowfit: value -9223372036854775809 is not in the domain of \"x\" (-9..-1)\n",
			wantFilePath: "m.model",
		},
		{
			name:         "satisfying assignment prints ok",
			args:         []string{"check", "m.model", "x=1", "y=2"},
			model:        argsCheckModel,
			wantCode:     0,
			wantStdout:   "ok\n",
			wantFilePath: "m.model",
		},
		{
			name:         "violations print all statements in source order",
			args:         []string{"check", "m.model", "x=2", "y=1"},
			model:        argsCheckModel,
			wantCode:     1,
			wantStdout:   "violated 3: require x < y\n",
			wantFilePath: "m.model",
		},
		{
			name:         "objective does not affect assignment checking",
			args:         []string{"check", "m.model", "x=2"},
			model:        "var x 1..2\nminimize x\n",
			wantCode:     0,
			wantStdout:   "ok\n",
			wantFilePath: "m.model",
		},
		{
			name:         "original path is read unchanged",
			args:         []string{"fmt", "a\\b.model"},
			model:        "var x 1..2\n",
			wantCode:     0,
			wantStdout:   "var x 1..2\n",
			wantFilePath: "a\\b.model",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			argsAssertCase(t, tc)
		})
	}
}

func TestArgsCheckMissingModelAndMalformedBeforeLoad(t *testing.T) {
	cases := []argsContractCase{
		{
			name:       "check with no model",
			args:       []string{"check"},
			wantCode:   2,
			wantStderr: "narrowfit: missing MODEL\n",
		},
		{
			name:       "assignment syntax is checked before any file read",
			args:       []string{"check", "not-read.model", "x=+3"},
			wantCode:   2,
			wantStderr: "narrowfit: invalid assignment \"x=+3\"\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			argsAssertCase(t, tc)
		})
	}
}

var _ io.Reader = (*argsTrackingReader)(nil)
