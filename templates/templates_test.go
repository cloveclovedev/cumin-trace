package templates

import (
	"bytes"
	"io/fs"
	"os"
	"path"
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

	// The directories that site/.gitignore names hold what npm and Astro
	// write, not files of the template.
	ignored := map[string]bool{"site/node_modules": true, "site/dist": true, "site/.astro": true}
	err = fs.WalkDir(os.DirFS(".."), "site", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if ignored[name] {
				return fs.SkipDir
			}
			return nil
		}
		if !templated[name] {
			t.Errorf("%s has no template in templates/site/", name)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
