package templates

import (
	"bytes"
	"io/fs"
	"os"
	"path"
	"testing"
)

// CTR-SITE-001: the site of this repository is the site template that init
// writes, byte for byte. The template names node-version and gitignore
// without the leading ".".
func TestTheSiteOfThisRepositoryIsTheTemplate(t *testing.T) {
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
}
