<p align="center"><img src="docs/logo.svg" width="96" alt="" /></p>

# narrowfit

[![CI](https://github.com/Gulliby-development/narrowfit/actions/workflows/ci.yml/badge.svg)](https://github.com/Gulliby-development/narrowfit/actions/workflows/ci.yml) [![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

narrowfit is an offline command-line tool for small combinatorial problems. It helps people who need to model finite integer choices and their relationships—such as schedules, assignments, or puzzles—and then find, enumerate, check, or explain solutions. It uses deterministic search and the Go standard library; it does not need a network connection.

## Install

Install Go 1.27.0, then build the program from the repository root:

```sh
go build
```

The build produces `narrowfit` on macOS and Linux and `narrowfit.exe` on Windows. The examples use the model `magic.model` shown under Usage: save that text as `magic.model` in the repository root, then run the executable from there:

```sh
./narrowfit solve magic.model
```

On Windows, use:

```powershell
.\narrowfit.exe solve magic.model
```

## Usage

The command-line forms are:

```text
usage: narrowfit solve [--json] MODEL
       narrowfit all [--limit N] MODEL
       narrowfit count [--limit N] MODEL
       narrowfit check MODEL NAME=VALUE...
       narrowfit explain MODEL
       narrowfit fmt [--check] MODEL
```

A model is read from the path given as `MODEL`; use `-` to read it from standard input. The complete `magic.model` and the result of `solve magic.model` are shown here.

### `magic.model`

```text
# 3x3 magic square
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
```

### `solve magic.model`

```text
a = 2
b = 7
c = 6
d = 9
e = 5
f = 1
g = 4
h = 3
i = 8
```

## Model language

A model is UTF-8 text with one statement per line. Blank lines are ignored, and `#` starts a comment through the end of its line. Declare each variable with a finite integer domain; domains may list individual values or inclusive ranges, and are normalized into ascending order.

The six constraint statements are:

- `alldiff NAME NAME...` requires the listed variables to have different values.
- `sum NAME... OP INT` compares the sum of the listed values with an integer.
- `count INT NAME... OP INT` compares how many listed variables equal the first integer with the final integer.
- `table NAME... : TUPLE | TUPLE...` restricts the listed values to one of the allowed tuples.
- `element NAME in NAME... is NAME` selects a listed variable by a one-based index and equates its value with the result variable.
- `require EXPR` requires an expression to hold.

Expressions support integer arithmetic, comparisons, logical operators, parentheses, and the `abs`, `min`, and `max` functions. Use `minimize EXPR` or `maximize EXPR` to select an objective. Statements and expressions are canonicalized with consistent spacing and only the parentheses needed to preserve their meaning; `fmt` prints that canonical form.

## Commands

- `solve [--json] MODEL` prints the best assignment, or the first assignment when there is no objective. It prints `no solution` when there is no eligible solution. With `--json` it prints one line of JSON instead (see [JSON output](#json-output)).
- `all [--limit N] MODEL` lists solutions in lexicographic order of their values in variable declaration order.
- `count [--limit N] MODEL` counts solutions without listing their assignments.
- `check MODEL NAME=VALUE...` checks a complete assignment against variable domains and all constraints.
- `explain MODEL` reports whether the model is satisfiable or gives a deterministic minimal conflicting set of constraints.
- `fmt [--check] MODEL` prints the model's canonical statements. With `--check` it prints nothing but a verdict: `ok` and exit 0 when every statement line is already written in its canonical form, else one line `FILE:LINE: not canonical, expected TEXT` for each line that is not, a count, and exit 1. Blank lines, comments, line endings and a byte order mark are not compared.

For `all`, the default limit is 20; for `count`, it is 1000000. Both accept `--limit N` or `--limit=N`, where `N` is from 1 through 1000000. If the option occurs more than once, the last value is used. `all` and `count` report when additional solutions exist beyond the limit.

## JSON output

`solve --json` writes one line: an object whose member `solution` holds the variables in declaration order (`null` when there is no solution, with exit code 1) and, when the model has an objective, a member `objective` after it. There are no spaces. `--json` is an option of `solve` only. For a model that declares `var x 1..3`, `var y 1..3` and `maximize x - y`:

```sh
narrowfit solve --json model.txt
```

prints:

```text
{"solution":{"x":3,"y":1},"objective":2}
```

## Exit codes

- `0`: the command completed successfully.
- `1`: a negative result, such as no solution or a failed assignment check.
- `2`: a command-line, input, or model error; diagnostics are written to standard error.

## Development

From the repository root, format and check the project with:

```sh
gofmt -l .
go vet ./...
go test ./...
```

A successful `gofmt -l .` prints nothing. The vet and test commands should complete without errors.

---

Built with BAKED
