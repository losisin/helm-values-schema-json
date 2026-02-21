package pkg

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

type FS interface {
	fs.FS
	fs.StatFS
	fs.ReadDirFS
}

func ParseArgs(ctx context.Context, args []string, config *Config) ([]string, error) {
	root, err := os.OpenRoot(".")
	if err != nil {
		return nil, err
	}
	return ParseArgsFS(ctx, root.FS().(FS), args, config)
}

// ParseArgsFS takes the input CLI arguments and returns a list of directories.
//
// The inputs may use glob patterns (with double-star support).
func ParseArgsFS(ctx context.Context, fsys FS, args []string, config *Config) ([]string, error) {
	if !config.Recursive {
		if len(args) == 0 {
			return []string{"."}, nil
		}
		return args, nil
	}

	walker := NewWalker(fsys, config)

	if len(args) == 0 {
		args = []string{""}
	}

	for _, arg := range args {
		if err := walker.WalkArg(ctx, arg); err != nil {
			return nil, fmt.Errorf("walk directories using glob patterns: %w", err)
		}
	}

	if len(walker.Dirs) == 0 {
		return nil, fmt.Errorf("no matching directories found")
	}

	return walker.Dirs, nil
}

type Walker struct {
	Dirs    []string
	fsys    FS
	ignorer *GitIgnorer
	config  *Config
}

func NewWalker(fsys FS, config *Config) *Walker {
	ignorer := NewGitIgnorer(fsys)
	if config.NoGitIgnore {
		ignorer.Disable()
	}
	return &Walker{
		fsys:    fsys,
		ignorer: ignorer,
		config:  config,
	}
}

func (w *Walker) WalkArg(ctx context.Context, arg string) error {
	globOptions := []doublestar.GlobOption{
		doublestar.WithFailOnPatternNotExist(),
		doublestar.WithFailOnIOErrors(),
	}
	if !w.config.Hidden {
		globOptions = append(globOptions, doublestar.WithNoHidden())
	}
	return doublestar.GlobWalk(w.fsys, path.Join(filepath.ToSlash(arg), "**")+"/", func(path string, d fs.DirEntry) error {
		switch {
		case !d.IsDir(),
			slices.Contains(w.Dirs, path),
			w.ignorer.IsIgnored(ctx, path),
			!isValidDirectory(w.fsys, path, w.config):
			// skip
			return nil
		}
		w.Dirs = append(w.Dirs, path)
		return nil
	}, globOptions...)
}

func isValidDirectory(fsys FS, dir string, config *Config) bool {
	if len(config.RecursiveNeeds) == 0 || config.NoRecursiveNeeds {
		return true
	}
	entries, err := fsys.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, requiredFile := range config.RecursiveNeeds {
		if !entriesContains(entries, requiredFile) {
			return false
		}
	}
	return true
}

func entriesContains(entries []fs.DirEntry, basename string) bool {
	for _, entry := range entries {
		if filepath.Base(entry.Name()) == basename {
			return true
		}
	}
	return false
}

type GitIgnorer struct {
	fsys       FS
	ignoredMap map[string][]string
	disabled   bool
}

func NewGitIgnorer(fsys FS) *GitIgnorer {
	return &GitIgnorer{
		fsys:       fsys,
		ignoredMap: make(map[string][]string),
	}
}

func (g *GitIgnorer) Disable() {
	g.disabled = true
}

func (g *GitIgnorer) IsIgnored(ctx context.Context, path string) bool {
	if g.disabled {
		return false
	}
	path = filepath.Clean(path)
	ignored, ok := g.ignoredMap[path]
	if !ok {
		ignored = g.execGitIgnore(ctx, path)
		g.ignoredMap[path] = ignored
	}
	return slices.Contains(ignored, path)
}

func (g *GitIgnorer) execGitIgnore(ctx context.Context, path string) []string {
	cmd := exec.CommandContext(ctx, "git", "check-ignore", "-z", "--stdin")

	entries, err := g.fsys.ReadDir(path)
	if err != nil {
		g.disabled = true
		LoggerFromContext(ctx).Logf("Unexpected error when listing files in directory: %s", err)
		return nil
	}
	var stdin bytes.Buffer

	stdin.WriteString(path)
	for _, entry := range entries {
		if stdin.Len() > 0 {
			stdin.WriteByte(0)
		}
		stdin.WriteString(filepath.Join(path, entry.Name()))
	}
	cmd.Stdin = &stdin

	output, err := cmd.Output()
	if err != nil {
		var execErr *exec.ExitError
		if errors.As(err, &execErr) {
			if execErr.ExitCode() == 1 {
				// no files are ignored
				return nil
			}
			if execErr.ExitCode() == 128 {
				// fatal error in Git, such as we're not in a Git directory
				g.disabled = true
				LoggerFromContext(ctx).Logf("Error from Git when checking ignored files: %s",
					strings.ReplaceAll(string(execErr.Stderr), "\n", "\t"))
				return nil
			}
			if strings.Contains(execErr.Error(), "executable file not found") {
				g.disabled = true
				return nil
			}
		}
		g.disabled = true
		LoggerFromContext(ctx).Logf("Unexpected error when checking ignored files: %s", err)
		return nil
	}

	return slices.DeleteFunc(
		strings.Split(string(output), "\x00"),
		func(name string) bool { return name == "" },
	)
}
