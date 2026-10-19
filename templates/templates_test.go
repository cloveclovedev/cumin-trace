package templates

import (
	"bytes"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"
)

// CTR-SITE-001: the site of this repository is the site template that init
// writes, byte for byte, and holds no file that the template does not have.
// The template names node-version and gitignore without the leading ".".
func TestTheSiteOfThisRepositoryIsTheTemplate(t *testing.T) {
	templated := map[string]bool{}
	err := fs.WalkDir(Files, "site", func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		want, err := Files.ReadFile(name)
		if err != nil {
			return err
		}
		target := name
		if base := path.Base(name); base == "node-version" || base == "gitignore" {
			target = path.Join(path.Dir(name), "."+base)
		}
		templated[target] = true
		got, err := os.ReadFile(path.Join("..", target))
		if err != nil {
			t.Errorf("%s: %v", target, err)
			return nil
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s differs from templates/%s", target, name)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// The files of site/ that git tracks, or would track: git skips what npm
	// and Astro write, and any file that a .gitignore or the global excludes
	// of git name, such as .DS_Store.
	out, err := exec.Command("git", "-C", "..", "ls-files", "-z", "--cached", "--others", "--exclude-standard", "--", "site").Output()
	if err != nil {
		t.Fatalf("git ls-files: %v", err)
	}
	if len(out) == 0 {
		t.Fatal("git lists no file in site/")
	}
	for _, name := range strings.Split(strings.TrimSuffix(string(out), "\x00"), "\x00") {
		if !templated[name] {
			t.Errorf("%s has no template in templates/site/", name)
		}
	}
}
