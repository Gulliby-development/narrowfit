package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestREADMEHasRequiredSectionsAndExamples(t *testing.T) {
	data, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	readme := string(data)

	for _, heading := range []string{
		"## Install",
		"## Usage",
		"## Model language",
		"## Commands",
		"## Exit codes",
		"## Development",
	} {
		if !strings.Contains(readme, heading+"\n") {
			t.Errorf("README.md is missing heading %q", heading)
		}
	}

	usage := "```text\n" +
		"usage: narrowfit solve [--json] MODEL\n" +
		"       narrowfit all [--limit N] MODEL\n" +
		"       narrowfit count [--limit N] MODEL\n" +
		"       narrowfit check MODEL NAME=VALUE...\n" +
		"       narrowfit explain MODEL\n" +
		"       narrowfit fmt [--check] MODEL\n" +
		"```"
	if !strings.Contains(readme, usage) {
		t.Errorf("README.md does not contain the exact usage block")
	}

	magicModel := "### `magic.model`\n\n```text\n" +
		"# 3x3 magic square\n" +
		"var a 1..9\n" +
		"var b 1..9\n" +
		"var c 1..9\n" +
		"var d 1..9\n" +
		"var e 1..9\n" +
		"var f 1..9\n" +
		"var g 1..9\n" +
		"var h 1..9\n" +
		"var i 1..9\n" +
		"alldiff a b c d e f g h i\n" +
		"sum a b c == 15\n" +
		"sum d e f == 15\n" +
		"sum g h i == 15\n" +
		"sum a d g == 15\n" +
		"sum b e h == 15\n" +
		"sum c f i == 15\n" +
		"sum a e i == 15\n" +
		"sum c e g == 15\n" +
		"```"
	if !strings.Contains(readme, magicModel) {
		t.Errorf("README.md does not contain the exact magic.model block")
	}

	solveOutput := "### `solve magic.model`\n\n```text\n" +
		"a = 2\n" +
		"b = 7\n" +
		"c = 6\n" +
		"d = 9\n" +
		"e = 5\n" +
		"f = 1\n" +
		"g = 4\n" +
		"h = 3\n" +
		"i = 8\n" +
		"```"
	if !strings.Contains(readme, solveOutput) {
		t.Errorf("README.md does not contain the exact solve magic.model output")
	}

	if got := strings.Count(readme, "```"); got != 18 {
		t.Errorf("README.md has %d code fences, want 18 for the documented blocks", got)
	}
}

func TestExecutableRejectsMalformedUTF8OnStdin(t *testing.T) {
	executableName := "narrowfit"
	if runtime.GOOS == "windows" {
		executableName += ".exe"
	}
	executablePath := filepath.Join(t.TempDir(), executableName)

	buildContext, cancelBuild := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelBuild()
	build := exec.CommandContext(buildContext, "go", "build", "-o", executablePath, ".")
	buildOutput, err := build.CombinedOutput()
	if err != nil {
		t.Fatalf("go build -o %s . failed: %v\n%s", executablePath, err, buildOutput)
	}
	if err := buildContext.Err(); err != nil {
		t.Fatalf("go build -o %s . timed out: %v", executablePath, err)
	}

	runContext, cancelRun := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelRun()
	command := exec.CommandContext(runContext, executablePath, "solve", "-")
	command.Stdin = bytes.NewReader([]byte{
		0x76, 0x61, 0x72, 0x20, 0x78, 0x20, 0x31, 0x2e,
		0x2e, 0x32, 0x20, 0x23, 0x20, 0xff, 0x0a,
	})
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	err = command.Run()
	if err := runContext.Err(); err != nil {
		t.Fatalf("narrowfit solve - timed out: %v", err)
	}
	var exitError *exec.ExitError
	if !errors.As(err, &exitError) {
		t.Fatalf("narrowfit solve - error = %v, want exit status 2", err)
	}
	if got := exitError.ExitCode(); got != 2 {
		t.Errorf("narrowfit solve - exit code = %d, want 2", got)
	}
	if got := stdout.String(); got != "" {
		t.Errorf("narrowfit solve - stdout = %q, want empty", got)
	}
	if got, want := stderr.String(), "narrowfit: <stdin>: input is not valid UTF-8\n"; got != want {
		t.Errorf("narrowfit solve - stderr = %q, want %q", got, want)
	}
}
