
# Build Prompt — Enforce Testing Principles (1–7) for a Go CLI

You are a senior Go engineer implementing a cross-platform CLI. Your primary objective is to structure the code and tests so the program strictly adheres to the following **Principles 1–7** and ships with guardrails that enforce them in CI.

## Principles (must implement)

1. **Separate core from CLI**

   * All business logic lives in internal packages (e.g., `semver`, `policy`, `execx`, `config`).
   * The command layer only parses flags and calls internal packages.

2. **Testable exec runner**

   * Define an interface:

     ```go
     type Runner interface {
       Run(ctx context.Context, name string, args []string, env []string, wd string) (stdout, stderr string, exit int, err error)
     }
     ```
   * Provide a real implementation (uses `exec.CommandContext`) and a fake for unit tests.
   * Allow tests to place stub binaries on `PATH` to simulate external tools.

3. **No `os.Exit` in library code**

   * Commands return `error`.
   * Only `main()` maps errors to exit codes via a small helper:

     ```go
     func exitCode(err error) int { /* stable mapping */ }
     ```
   * Export `NewRootCmd()` (Cobra) or `NewApp()` (urfave/cli) to enable in-process command tests.

4. **Deterministic output**

   * Support `--json`, `--no-color`, and `--quiet` flags.
   * Disable TTY spinners when `CI=1` or when not a TTY.
   * Inject `clock` and `rand` where time/entropy is used.
   * Normalize path separators and line endings for tests.

5. **Filesystem strategy**

   * Use real FS in tests with `t.TempDir()`; keep FS-touching code isolated behind small functions.
   * Prefer no global CWD mutation; accept `--workdir` or inject WD in code paths.

6. **Golden tests**

   * Provide a `-update` flag in tests to refresh goldens.
   * Normalize `\r\n`→`\n`.
   * Store goldens under `testdata/golden/`.

7. **Testscript workflows**

   * Use `rogpeppe/go-internal/testscript` for end-to-end scenarios under `testdata/script/`.
   * Include a reusable `apx` command wrapper in the harness to run the built binary and assert exit codes/stdout/stderr.


## Implement These Tests

### A) Unit tests (table-driven)

* Packages: `semver`, `policy`, `config`, any pure logic.
* Use `testing` + `require` (from `stretchr/testify`) or `quicktest`.
* Add **fuzz tests** for config parsing (`go test` fuzzing).

### B) Command tests (in-process)

* Build command with `NewRootCmd()`.
* Inject `io.Writer` for stdout/stderr; set args with `cmd.SetArgs(...)`.
* Assert on output and returned error (no `os.Exit`).

### C) Integration tests (subprocess)

* Build the binary once per package (`go test` setup) and run with `os/exec` or `gotest.tools/icmd` (or `gomega/gexec`).
* Use `t.TempDir()` as workspace; control env (`PATH`, `HOME`, `CI`, `NO_COLOR`).
* Verify exit codes, stdout/stderr, and file outputs.

### D) Golden tests

* Provide helper:

  ```go
  var update = flag.Bool("update", false, "update golden files")
  ```
* Compare output with goldens under `testdata/golden/`. On mismatch, print unified diff (use `go-cmp`).

### E) Testscript scenarios

* Add at least two scripts:

  * `help.txt` – calls `cli-name help`, checks usage text.
  * `workflow_semver.txt` – simulates a tiny repo change and runs `cli-name semver suggest ...`.
* Harness should inject the built binary path and set `APX_DISABLE_TTY=1`.

---

## CI Requirements (GitHub Actions)

* OS matrix: `ubuntu-latest`, `macos-latest`, `windows-latest`.
* Steps:

  1. `go build -o ./bin/cli-name ./cmd/<cli-name>`
  2. `go test ./... -race -count=1 -coverprofile=coverage.out`
  3. Run e2e/testscript suites.
* **Gates that enforce Principles:**

  * **No `os.Exit`** outside `cmd/.../main.go` (grep check):

    ```bash
    ! git grep -n 'os\\.Exit' -- ':!cmd/**/main.go'
    ```
  * **No direct `exec.Command`** outside `internal/execx`:

    ```bash
    ! git grep -n 'exec\\.Command' -- ':!internal/execx/**'
    ```
  * **Colorless CI**: set `CI=1` and `NO_COLOR=1` env; tests must pass with identical outputs across OSes.

---

## Acceptance Criteria (must pass)

* `NewRootCmd()` exists; `main()` only maps errors to exit codes.
* Unit tests cover core logic; fuzz tests do not panic.
* Command tests succeed without forking a subprocess or calling `os.Exit`.
* Integration tests run the real binary and validate exit codes and artifacts.
* Golden tests pass; `-update` refreshes expected outputs.
* Testscript scenarios pass on all three OSes.
* CI grep checks confirm:

  * no `os.Exit` outside `main.go`,
  * no direct `exec.Command` outside `internal/execx`.

---

## Developer Ergonomics

* Global flags supported: `--json`, `--no-color`, `--quiet`, `--verbose`.
* All commands return stable JSON when `--json` is set.
* Spinners/ANSI disabled automatically in CI or with `--no-color`.

---

## Deliverables

* Compilable Go module with the structure above.
* Full test suite (`go test ./... -race`) passing on Linux/macOS/Windows.
* CI workflow file that enforces the grep gates and runs e2e + testscript.
* `CONTRIBUTING.md` explaining how to run unit, e2e, golden, and testscript tests.

> **Do not proceed** unless each Principle (1–7) is demonstrably enforced by tests and CI gates.
