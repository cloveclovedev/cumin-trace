// Command cumin-trace reads the requirement documents of a product and shows
// how far each requirement has come.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/cloveclovedev/cumin-trace/internal/check"
)

func main() {
	os.Exit(run(os.Args[1:], ".", os.Stdout, os.Stderr))
}

// run chooses the command from the arguments and returns the exit code
// (CTR-CHK-003): 0 with no error, 1 with a format error, and 2 when the
// command cannot run.
func run(args []string, root string, stdout, stderr io.Writer) int {
	if len(args) != 1 || args[0] != "check" {
		fmt.Fprintln(stderr, "cumin-trace: usage: cumin-trace check")
		return 2
	}
	count, err := check.Run(root, stdout)
	if err != nil {
		fmt.Fprintf(stderr, "cumin-trace: %v\n", err)
		return 2
	}
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
