// Command cumin-trace reads the requirement documents of a product and shows
// how far each requirement has come.
package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime/debug"
	"strings"
	"time"

	"github.com/cloveclovedev/cumin-trace/internal/check"
	"github.com/cloveclovedev/cumin-trace/internal/export"
	"github.com/cloveclovedev/cumin-trace/internal/scaffold"
	"github.com/cloveclovedev/cumin-trace/internal/trace"
)

const usage = "cumin-trace: usage: cumin-trace check | cumin-trace export | cumin-trace init <prefix> [<area>] | cumin-trace trace | cumin-trace version"

// version is the version of the binary. A release build sets it with
// -ldflags "-X main.version=<tag>".
var version string

func main() {
	os.Exit(run(os.Args[1:], ".", os.Stdout, os.Stderr))
}

// traceEnv reads the environment of the trace. Only this function reads the
// environment variables and runs git remote. The trace runs git too, to read
// the history (internal/trace/history.go). A failure of git leaves Remote
// empty.
func traceEnv(root string) trace.Env {
	env := trace.Env{
		Token:      os.Getenv("GITHUB_TOKEN"),
		Repository: os.Getenv("GITHUB_REPOSITORY"),
		GraphQLURL: os.Getenv("GITHUB_GRAPHQL_URL"),
		Now:        time.Now,
	}
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = root
	if out, err := cmd.Output(); err == nil {
		env.Remote = strings.TrimSpace(string(out))
	}
	return env
}

// run chooses the command from the arguments and returns the exit code.
func run(args []string, root string, stdout, stderr io.Writer) int {
	switch {
	case len(args) == 1 && args[0] == "check":
		return runCheck(root, stdout, stderr)
	case len(args) == 1 && args[0] == "export":
		return runExport(root, stdout, stderr)
	case len(args) > 0 && args[0] == "init":
		return runInit(args[1:], root, stdout, stderr)
	case len(args) == 1 && args[0] == "trace":
		return runTrace(root, traceEnv(root), stdout, stderr)
	case len(args) == 1 && args[0] == "version":
		return runVersion(stdout)
	}
	fmt.Fprintln(stderr, usage)
	return 2
}

// runCheck runs the format check and returns the exit code (CTR-CHK-003): 0
// with no error, 1 with a format error, and 2 when the check cannot run.
func runCheck(root string, stdout, stderr io.Writer) int {
	count, err := check.Run(root, stdout)
	if err != nil {
		fmt.Fprintf(stderr, "cumin-trace: %v\n", err)
		return 2
	}
	return countExit(count, stderr)
}

// countExit prints the number of the format errors to stderr and returns 1
// when there is an error, and 0 when there is none.
func countExit(count int, stderr io.Writer) int {
	if count == 1 {
		fmt.Fprintln(stderr, "cumin-trace: 1 error")
		return 1
	}
	if count > 1 {
		fmt.Fprintf(stderr, "cumin-trace: %d errors\n", count)
		return 1
	}
	return 0
}

// runExport writes every requirement as JSON to stdout (CTR-EXP-001). It
// returns 0 when the JSON is written, 1 with a format error (CTR-EXP-003),
// and 2 when the export cannot run.
func runExport(root string, stdout, stderr io.Writer) int {
	count, err := export.Run(root, stdout, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "cumin-trace: %v\n", err)
		return 2
	}
	return countExit(count, stderr)
}

// runInit writes the templates and returns the exit code (CTR-TPL-001): 0
// when every file is written or skipped, and 2 when the arguments are wrong
// or a file cannot be written.
func runInit(args []string, root string, stdout, stderr io.Writer) int {
	if len(args) != 1 && len(args) != 2 {
		fmt.Fprintln(stderr, "cumin-trace: usage: cumin-trace init <prefix> [<area>]")
		return 2
	}
	prefix, area := args[0], ""
	if len(args) == 2 {
		area = args[1]
		// An empty argument is a wrong area, not a missing one.
		if err := scaffold.CheckArea(area); err != nil {
			fmt.Fprintf(stderr, "cumin-trace: %v\n", err)
			return 2
		}
	}
	if err := scaffold.Run(root, prefix, area, binaryVersion(), stdout); err != nil {
		fmt.Fprintf(stderr, "cumin-trace: %v\n", err)
		return 2
	}
	return 0
}

// runTrace links the requirements to the requirement issues and writes the
// JSON to stdout (CTR-TRC-008). It returns 0 when the JSON is written, 1 with
// a format error (CTR-TRC-009), and 2 when the trace cannot run.
func runTrace(root string, env trace.Env, stdout, stderr io.Writer) int {
	count, err := trace.Run(root, env, stdout, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "cumin-trace: %v\n", err)
		return 2
	}
	return countExit(count, stderr)
}

// runVersion prints the version of the binary in one line and returns 0
// (CTR-CMD-002). A build without a version prints "(devel)".
func runVersion(stdout io.Writer) int {
	v := binaryVersion()
	if v == "" {
		v = "(devel)"
	}
	fmt.Fprintln(stdout, v)
	return 0
}

// binaryVersion returns the version that the build set, or else the version
// of the main module that Go records in the binary.
func binaryVersion() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		return info.Main.Version
	}
	return ""
}
