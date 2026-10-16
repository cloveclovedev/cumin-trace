package scaffold

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// CTR-TPL-006: after init, the script of the diagrams has mode 0755 and the
// Dockerfile of its image has mode 0644.
func TestWritesTheScriptOfTheDiagramsAsAnExecutable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows has no permission bits of Unix")
	}
	root := t.TempDir()
	runInit(t, root, "PC", "")

	for name, want := range map[string]os.FileMode{RenderScript: 0o755, "tools/plantuml/Dockerfile": 0o644} {
		info, err := os.Stat(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != want {
			t.Errorf("%s has mode %v, want %v", name, got, want)
		}
	}
}

// CTR-TPL-006: the script that init writes prints its usage and exits with 2
// for a wrong count of arguments. It stops before it uses Docker.
func TestTheScriptOfTheDiagramsPrintsItsUsageForAWrongArgumentCount(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the script runs with /bin/sh")
	}
	root := t.TempDir()
	runInit(t, root, "PC", "")

	for _, args := range [][]string{{"one"}, {"one", "two", "three"}} {
		var stderr strings.Builder
		cmd := exec.Command(filepath.Join(root, filepath.FromSlash(RenderScript)), args...)
		cmd.Stderr = &stderr

		err := cmd.Run()

		var exit *exec.ExitError
		if !errors.As(err, &exit) || exit.ExitCode() != 2 {
			t.Errorf("with %d arguments: got error %v, want exit code 2", len(args), err)
		}
		for _, use := range []string{"scripts/render-diagrams.sh <in.puml> <out.svg>", "scripts/render-diagrams.sh --check <base>"} {
			if !strings.Contains(stderr.String(), use) {
				t.Errorf("with %d arguments: the usage has no line %q:\n%s", len(args), use, stderr.String())
			}
		}
	}
}

// CTR-TPL-006: the script and the Dockerfile name the same tag of the
// PlantUML image, so that the script renders with the image that the
// Dockerfile builds.
func TestTheScriptAndTheDockerfileNameTheSameImageTag(t *testing.T) {
	root := t.TempDir()
	runInit(t, root, "PC", "")

	script := regexp.MustCompile(`(?m)^image="cumin-plantuml:([0-9.]+)"$`).FindStringSubmatch(readFile(t, root, RenderScript))
	dockerfile := regexp.MustCompile(`(?m)^FROM plantuml/plantuml:([0-9.]+)$`).FindStringSubmatch(readFile(t, root, "tools/plantuml/Dockerfile"))
	if script == nil || dockerfile == nil {
		t.Fatalf("got script tag %q and Dockerfile tag %q, want a tag in each", script, dockerfile)
	}
	if script[1] != dockerfile[1] {
		t.Errorf("the script names tag %s, the Dockerfile names tag %s; want the same", script[1], dockerfile[1])
	}
}
