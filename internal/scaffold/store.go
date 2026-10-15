package scaffold

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// create writes the content to a new file at path, and the directories above
// it that are missing. It creates the file in one operation that fails when
// the file exists, so that it never overwrites a file (CTR-TPL-001). It
// returns false with no error when the file exists. When the write or the
// close fails, it removes the file, so that the next run writes it again.
// The new file gets the mode as given, whatever the umask is.
func create(path string, content []byte, mode fs.FileMode) (bool, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if errors.Is(err, fs.ErrExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := file.Chmod(mode); err != nil {
		file.Close()
		os.Remove(path)
		return false, err
	}
	if _, err := file.Write(content); err != nil {
		file.Close()
		os.Remove(path)
		return false, err
	}
	if err := file.Close(); err != nil {
		os.Remove(path)
		return false, err
	}
	return true, nil
}

// exists reports whether a file or a directory is at path.
func exists(path string) (bool, error) {
	_, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}
