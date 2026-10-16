package cli_test

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/Gulliby-development/narrowfit/internal/cli"
)

type fmtCheckRunResult struct {
	code       int
	stdout     string
	stderr     string
	fileReads  int
	filePath   string
	stdinReads int
}

type fmtCheckTrackingReader struct {
	reader io.Reader
	reads  int
}

// Read counts reads while delegating to the wrapped reader.
func (reader *fmtCheckTrackingReader) Read(buffer []byte) (int, error) {
	reader.reads++
	return reader.reader.Read(buffer)
}

type fmtCheckFailingReader struct{}

// Read returns a deterministic error for stdin read-failure examples.
func (fmtCheckFailingReader) Read([]byte) (int, error) {
	return 0, errors.New("stdin failure")
}

// fmtCheckRun executes cli.Run with in-memory stdin and file providers.
func fmtCheckRun(args []string, stdin io.Reader, fileData []byte, fileErr error) fmtCheckRunResult {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	trackingStdin := &fmtCheckTrackingReader{reader: stdin}
	result := fmtCheckRunResult{}
	result.code = cli.Run(args, trackingStdin, &stdout, &stderr, func(path string) ([]byte, error) {
		result.fileReads++
		result.filePath = path
		return fileData, fileErr
	})
	result.stdout = stdout.String()
	result.stderr = stderr.String()
	result.stdinReads = trackingStdin.reads
	return result
}

// fmtCheckAssertResult compares a CLI result with its exact expected streams.
func fmtCheckAssertResult(t *testing.T, got fmtCheckRunResult, wantCode int, wantStdout, wantStderr string) {
	t.Helper()
	if got.code != wantCode || got.stdout != wantStdout || got.stderr != wantStderr {
		t.Fatalf("Run() = (code %d, stdout %q, stderr %q), want (code %d, stdout %q, stderr %q)",
			got.code, got.stdout, got.stderr, wantCode, wantStdout, wantStderr)
	}
}

// fmtCheckAssertFileRead verifies a single file-provider call with the original path.
func fmtCheckAssertFileRead(t *testing.T, got fmtCheckRunResult, wantPath string) {
	t.Helper()
	if got.fileReads != 1 || got.filePath != wantPath {
		t.Fatalf("readFile calls/path = (%d, %q), want (1, %q)", got.fileReads, got.filePath, wantPath)
	}
}

// fmtCheckAssertNoInputRead verifies argument errors do not access either input provider.
func fmtCheckAssertNoInputRead(t *testing.T, got fmtCheckRunResult) {
	t.Helper()
	if got.fileReads != 0 || got.stdinReads != 0 {
		t.Fatalf("input reads = (file %d, stdin %d), want none", got.fileReads, got.stdinReads)
	}
}

// fmtCheckAssertStdinRead verifies stdin was read without invoking the file provider.
func fmtCheckAssertStdinRead(t *testing.T, got fmtCheckRunResult) {
	t.Helper()
	if got.fileReads != 0 || got.stdinReads == 0 {
		t.Fatalf("input reads = (file %d, stdin %d), want stdin only", got.fileReads, got.stdinReads)
	}
}

// TestFmtCheckVerdictsAndArgumentPlacement covers verdicts, reports, and flag placement.
func TestFmtCheckVerdictsAndArgumentPlacement(t *testing.T) {
	const tidyModel = "# sizes\nvar x 5..7\nvar y 1..3 # keep\n\nsum x y >= 9\nrequire x > y\n"
	const messyModel = "# sizes\nvar   x   7,5,6\nvar y 1..3   # keep\nsum x  y >= 9\nrequire x>y\n"
	const messyReport = "messy.model:2: not canonical, expected var x 5..7\n" +
		"messy.model:4: not canonical, expected sum x y >= 9\n" +
		"messy.model:5: not canonical, expected require x > y\n" +
		"3 lines not canonical\n"

	fileCases := []struct {
		name       string
		args       []string
		path       string
		model      string
		wantCode   int
		wantStdout string
	}{
		{
			name:       "tidy model",
			args:       []string{"fmt", "--check", "tidy.model"},
			path:       "tidy.model",
			model:      tidyModel,
			wantCode:   0,
			wantStdout: "ok\n",
		},
		{
			name:       "flag follows model",
			args:       []string{"fmt", "tidy.model", "--check"},
			path:       "tidy.model",
			model:      tidyModel,
			wantCode:   0,
			wantStdout: "ok\n",
		},
		{
			name:       "repeated flag is harmless",
			args:       []string{"fmt", "--check", "--check", "tidy.model"},
			path:       "tidy.model",
			model:      tidyModel,
			wantCode:   0,
			wantStdout: "ok\n",
		},
		{
			name:       "messy model exact report",
			args:       []string{"fmt", "--check", "messy.model"},
			path:       "messy.model",
			model:      messyModel,
			wantCode:   1,
			wantStdout: messyReport,
		},
		{
			name:       "flag follows messy model",
			args:       []string{"fmt", "messy.model", "--check"},
			path:       "messy.model",
			model:      messyModel,
			wantCode:   1,
			wantStdout: messyReport,
		},
		{
			name:       "singular mismatch count",
			args:       []string{"fmt", "--check", "single.model"},
			path:       "single.model",
			model:      "var  x 1..2\n",
			wantCode:   1,
			wantStdout: "single.model:1: not canonical, expected var x 1..2\n1 line not canonical\n",
		},
	}
	for _, testCase := range fileCases {
		t.Run(testCase.name, func(t *testing.T) {
			got := fmtCheckRun(testCase.args, bytes.NewReader(nil), []byte(testCase.model), nil)
			fmtCheckAssertResult(t, got, testCase.wantCode, testCase.wantStdout, "")
			fmtCheckAssertFileRead(t, got, testCase.path)
		})
	}

	stdinCases := []struct {
		name       string
		args       []string
		wantCode   int
		wantStdout string
	}{
		{
			name:       "flag before stdin path",
			args:       []string{"fmt", "--check", "-"},
			wantCode:   1,
			wantStdout: "<stdin>:1: not canonical, expected var x 1..2\n1 line not canonical\n",
		},
		{
			name:       "flag after stdin path",
			args:       []string{"fmt", "-", "--check"},
			wantCode:   1,
			wantStdout: "<stdin>:1: not canonical, expected var x 1..2\n1 line not canonical\n",
		},
	}
	for _, testCase := range stdinCases {
		t.Run(testCase.name, func(t *testing.T) {
			got := fmtCheckRun(testCase.args, bytes.NewReader([]byte("var  x 1..2\n")), nil, nil)
			fmtCheckAssertResult(t, got, testCase.wantCode, testCase.wantStdout, "")
			fmtCheckAssertStdinRead(t, got)
		})
	}
}

// TestFmtCheckSourceNormalizationAndDifferences covers ignored source details and significant spacing.
func TestFmtCheckSourceNormalizationAndDifferences(t *testing.T) {
	t.Run("BOM CRLF comments blanks trailing whitespace and missing final LF are ignored", func(t *testing.T) {
		model := []byte("\xef\xbb\xbf# header \t\r\n\r\nvar x 1..2 \t# declaration\r\n# another comment\r\nrequire x > 0 \t")
		got := fmtCheckRun([]string{"fmt", "--check", "portable.model"}, bytes.NewReader(nil), model, nil)
		fmtCheckAssertResult(t, got, 0, "ok\n", "")
		fmtCheckAssertFileRead(t, got, "portable.model")
	})

	cases := []struct {
		name       string
		path       string
		model      []byte
		wantStdout string
	}{
		{
			name:       "leading indentation is significant",
			path:       "indent.model",
			model:      []byte("\tvar x 1..2\n"),
			wantStdout: "indent.model:1: not canonical, expected var x 1..2\n1 line not canonical\n",
		},
		{
			name:       "interior tab is significant",
			path:       "tab.model",
			model:      []byte("var\tx 1..2\n"),
			wantStdout: "tab.model:1: not canonical, expected var x 1..2\n1 line not canonical\n",
		},
		{
			name:       "doubled interior spaces are significant",
			path:       "spaces.model",
			model:      []byte("var x 1..2\nrequire  x > 0\n"),
			wantStdout: "spaces.model:2: not canonical, expected require x > 0\n1 line not canonical\n",
		},
		{
			name:       "unordered domain is significant",
			path:       "domain.model",
			model:      []byte("var x 7,5,6\n"),
			wantStdout: "domain.model:1: not canonical, expected var x 5..7\n1 line not canonical\n",
		},
		{
			name:       "compact operator is significant",
			path:       "operator.model",
			model:      []byte("var x 1..2\nrequire x>0\n"),
			wantStdout: "operator.model:2: not canonical, expected require x > 0\n1 line not canonical\n",
		},
		{
			name:       "ignored lines retain physical line numbers",
			path:       "lines.model",
			model:      []byte("# header\n\nvar x 1..2\n\n require x>0 # trailing\n"),
			wantStdout: "lines.model:5: not canonical, expected require x > 0\n1 line not canonical\n",
		},
		{
			name:       "backslashes are normalized in diagnostics",
			path:       `dir\m.model`,
			model:      []byte("var  x 1..2\n"),
			wantStdout: "dir/m.model:1: not canonical, expected var x 1..2\n1 line not canonical\n",
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got := fmtCheckRun([]string{"fmt", "--check", testCase.path}, bytes.NewReader(nil), testCase.model, nil)
			fmtCheckAssertResult(t, got, 1, testCase.wantStdout, "")
			fmtCheckAssertFileRead(t, got, testCase.path)
		})
	}
}

// TestFmtCheckErrorsAndArgumentValidation checks error order, parsing, and input failures.
func TestFmtCheckErrorsAndArgumentValidation(t *testing.T) {
	argumentCases := []struct {
		name       string
		args       []string
		wantStderr string
	}{
		{
			name:       "missing model",
			args:       []string{"fmt", "--check"},
			wantStderr: "narrowfit: missing MODEL\n",
		},
		{
			name:       "check option cannot take a value",
			args:       []string{"fmt", "--check=1", "tidy.model"},
			wantStderr: "narrowfit: unknown option \"--check=1\"\n",
		},
		{
			name:       "extra model is rejected",
			args:       []string{"fmt", "--check", "one.model", "two.model"},
			wantStderr: "narrowfit: too many arguments\n",
		},
		{
			name:       "unknown option precedes missing model",
			args:       []string{"fmt", "--unknown"},
			wantStderr: "narrowfit: unknown option \"--unknown\"\n",
		},
		{
			name:       "unknown option is found before excess arguments",
			args:       []string{"fmt", "one.model", "two.model", "--unknown"},
			wantStderr: "narrowfit: unknown option \"--unknown\"\n",
		},
	}
	for _, command := range []string{"solve", "all", "count", "check", "explain"} {
		argumentCases = append(argumentCases, struct {
			name       string
			args       []string
			wantStderr string
		}{
			name:       command + " rejects fmt-only option",
			args:       []string{command, "--check", "m.model"},
			wantStderr: "narrowfit: unknown option \"--check\"\n",
		})
	}
	for _, testCase := range argumentCases {
		t.Run(testCase.name, func(t *testing.T) {
			got := fmtCheckRun(testCase.args, bytes.NewReader(nil), []byte("var x 1..2\n"), nil)
			fmtCheckAssertResult(t, got, 2, "", testCase.wantStderr)
			fmtCheckAssertNoInputRead(t, got)
		})
	}

	invalidUTF8 := []byte{'v', 'a', 'r', ' ', 'x', ' ', '1', '.', '.', '2', ' ', '#', ' ', 0xff, '\n'}
	fileErrors := []struct {
		name       string
		path       string
		model      []byte
		readErr    error
		wantStderr string
	}{
		{
			name:       "bad domain keeps its source line",
			path:       "bad.model",
			model:      []byte("var x 1..3\nvar y 3..1\n"),
			wantStderr: "narrowfit: bad.model:2: invalid domain \"3..1\"\n",
		},
		{
			name:       "model with no variables is rejected",
			path:       "empty.model",
			model:      []byte("# no variables\n\t # comment\n"),
			wantStderr: "narrowfit: empty.model: no variables\n",
		},
		{
			name:       "invalid UTF-8 bytes are rejected",
			path:       "invalid.model",
			model:      invalidUTF8,
			wantStderr: "narrowfit: invalid.model: input is not valid UTF-8\n",
		},
		{
			name:       "read failure keeps the existing diagnostic",
			path:       "unavailable.model",
			readErr:    errors.New("unavailable"),
			wantStderr: "narrowfit: unavailable.model: cannot read the file\n",
		},
		{
			name:       "parse error suppresses earlier formatting differences",
			path:       "invalid-after-mismatch.model",
			model:      []byte("var  x 1..2\nvar y 3..1\n"),
			wantStderr: "narrowfit: invalid-after-mismatch.model:2: invalid domain \"3..1\"\n",
		},
	}
	for _, testCase := range fileErrors {
		t.Run(testCase.name, func(t *testing.T) {
			got := fmtCheckRun([]string{"fmt", "--check", testCase.path}, bytes.NewReader(nil), testCase.model, testCase.readErr)
			fmtCheckAssertResult(t, got, 2, "", testCase.wantStderr)
			fmtCheckAssertFileRead(t, got, testCase.path)
		})
	}

	t.Run("invalid UTF-8 stdin bytes are rejected by the real input path", func(t *testing.T) {
		got := fmtCheckRun([]string{"fmt", "--check", "-"}, bytes.NewReader(invalidUTF8), nil, nil)
		fmtCheckAssertResult(t, got, 2, "", "narrowfit: <stdin>: input is not valid UTF-8\n")
		fmtCheckAssertStdinRead(t, got)
	})

	t.Run("stdin read failure keeps the existing diagnostic", func(t *testing.T) {
		got := fmtCheckRun([]string{"fmt", "--check", "-"}, fmtCheckFailingReader{}, nil, nil)
		fmtCheckAssertResult(t, got, 2, "", "narrowfit: <stdin>: cannot read the file\n")
		fmtCheckAssertStdinRead(t, got)
	})
}

// TestFmtCheckPreservesFmtHelpAndUnsatisfiableBehavior checks unaffected fmt and help behavior.
func TestFmtCheckPreservesFmtHelpAndUnsatisfiableBehavior(t *testing.T) {
	t.Run("ordinary fmt still prints canonical statements", func(t *testing.T) {
		got := fmtCheckRun([]string{"fmt", "ordinary.model"}, bytes.NewReader(nil), []byte("var   x   2,1\n"), nil)
		fmtCheckAssertResult(t, got, 0, "var x 1..2\n", "")
		fmtCheckAssertFileRead(t, got, "ordinary.model")
	})

	t.Run("standalone help has the updated exact usage", func(t *testing.T) {
		got := fmtCheckRun([]string{"--help"}, bytes.NewReader(nil), nil, nil)
		want := "usage: narrowfit solve MODEL\n" +
			"       narrowfit all [--limit N] MODEL\n" +
			"       narrowfit count [--limit N] MODEL\n" +
			"       narrowfit check MODEL NAME=VALUE...\n" +
			"       narrowfit explain MODEL\n" +
			"       narrowfit fmt [--check] MODEL\n"
		fmtCheckAssertResult(t, got, 0, want, "")
		fmtCheckAssertNoInputRead(t, got)
	})

	t.Run("canonical unsatisfiable model is checked without solving", func(t *testing.T) {
		model := []byte("var x 1\nrequire x == 2\n")
		got := fmtCheckRun([]string{"fmt", "--check", "unsat.model"}, bytes.NewReader(nil), model, nil)
		fmtCheckAssertResult(t, got, 0, "ok\n", "")
		fmtCheckAssertFileRead(t, got, "unsat.model")
	})

	t.Run("noncanonical unsatisfiable model reports only formatting differences", func(t *testing.T) {
		model := []byte("var  x 1\nrequire x==2\n")
		got := fmtCheckRun([]string{"fmt", "--check", "unsat.model"}, bytes.NewReader(nil), model, nil)
		want := "unsat.model:1: not canonical, expected var x 1\n" +
			"unsat.model:2: not canonical, expected require x == 2\n" +
			"2 lines not canonical\n"
		fmtCheckAssertResult(t, got, 1, want, "")
		fmtCheckAssertFileRead(t, got, "unsat.model")
	})
}
