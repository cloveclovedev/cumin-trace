package trace

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cloveclovedev/cumin-trace/internal/requirement"
)

// errShallow says that the git history is not deep enough (CTR-TRC-010).
var errShallow = errors.New("shallow git repository: fetch the whole history of the requirement documents (fetch-depth: 0)")

// errNoRepository says that root is in no git repository (CTR-TRC-010).
var errNoRepository = errors.New("no git repository: the trace reads the git history of the requirement documents")

// lastChangesOf reads the git history of the requirement documents under dir
// of the repository at root, and returns the time of the last change of each
// ID (CTR-TRC-010). A shallow repository, or no git repository, is an error.
func lastChangesOf(root, dir, prefix string) (map[string]time.Time, error) {
	commits, err := readHistory(root, dir, prefix)
	if err != nil {
		return nil, err
	}
	return LastChanges(commits), nil
}

// readHistory reads the commits that touch dir, oldest first, with the
// requirements of each commit and of its first parent. The path dir is
// relative to root, and root is the top of the repository.
func readHistory(root, dir, prefix string) ([]Commit, error) {
	shallow, err := runGit(root, nil, "rev-parse", "--is-shallow-repository")
	if err != nil {
		return nil, errNoRepository
	}
	if strings.TrimSpace(string(shallow)) == "true" {
		return nil, errShallow
	}
	if _, err := runGit(root, nil, "rev-parse", "--verify", "--quiet", "HEAD"); err != nil {
		// A repository without a commit has no history.
		return nil, nil
	}
	log, err := runGit(root, nil, "log", "--reverse", "--format=%x1e%H%x1f%P%x1f%cI%x1f%B", "--", dir)
	if err != nil {
		return nil, err
	}
	cache := map[string][]requirement.Requirement{}
	requirementsAt := func(commit string) ([]requirement.Requirement, error) {
		if reqs, ok := cache[commit]; ok {
			return reqs, nil
		}
		files, err := filesAt(root, dir, commit)
		if err != nil {
			return nil, err
		}
		reqs := requirement.Parse(files, prefix).Requirements
		cache[commit] = reqs
		return reqs, nil
	}
	var commits []Commit
	for _, record := range strings.Split(string(log), "\x1e")[1:] {
		fields := strings.SplitN(record, "\x1f", 4)
		if len(fields) != 4 {
			return nil, errors.New("cannot read the git history")
		}
		when, err := time.Parse(time.RFC3339, fields[2])
		if err != nil {
			return nil, fmt.Errorf("cannot read the time of the commit %s", fields[0])
		}
		commit := Commit{Time: when, Message: fields[3]}
		if commit.After, err = requirementsAt(fields[0]); err != nil {
			return nil, err
		}
		if parents := strings.Fields(fields[1]); len(parents) > 0 {
			if commit.Before, err = requirementsAt(parents[0]); err != nil {
				return nil, err
			}
		}
		commits = append(commits, commit)
	}
	return commits, nil
}

// filesAt returns the Markdown files below dir at the commit, at any depth,
// in byte order of the path, as requirement.Read reads them on the disk. It
// takes no symbolic link.
func filesAt(root, dir, commit string) ([]requirement.File, error) {
	tree, err := runGit(root, nil, "ls-tree", "-r", "-z", commit, "--", dir)
	if err != nil {
		return nil, err
	}
	var files []requirement.File
	var objects bytes.Buffer
	for _, entry := range strings.Split(string(tree), "\x00") {
		// An entry is "<mode> <type> <object>\t<path>".
		info, path, ok := strings.Cut(entry, "\t")
		fields := strings.Fields(info)
		if !ok || len(fields) != 3 || fields[1] != "blob" || fields[0] == "120000" || !strings.HasSuffix(path, ".md") {
			continue
		}
		files = append(files, requirement.File{Path: path})
		objects.WriteString(fields[2] + "\n")
	}
	if len(files) == 0 {
		return nil, nil
	}
	out, err := runGit(root, &objects, "cat-file", "--batch")
	if err != nil {
		return nil, err
	}
	// Each object is "<object> <type> <size>\n<content>\n", in the order of
	// the input.
	for i := range files {
		header, rest, ok := bytes.Cut(out, []byte("\n"))
		fields := strings.Fields(string(header))
		if !ok || len(fields) != 3 {
			return nil, errors.New("cannot read the git history")
		}
		size, err := strconv.Atoi(fields[2])
		if err != nil || size+1 > len(rest) {
			return nil, errors.New("cannot read the git history")
		}
		files[i].Source = rest[:size]
		out = rest[size+1:]
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

// runGit runs git in root with the input and returns the standard output.
func runGit(root string, input *bytes.Buffer, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	if input != nil {
		cmd.Stdin = input
	}
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("cannot read the git history: git %s failed", args[0])
	}
	return out, nil
}
