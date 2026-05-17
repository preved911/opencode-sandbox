// Package paths resolves user-facing path strings.
//
// The three forms shown in the project spec all converge to the same
// absolute path:
//
//	/Users/me/path/to/file
//	~/path/to/file
//	$HOME/path/to/file
package paths

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// Expand resolves p in this order:
//
//  1. environment substitution ($VAR, ${VAR}) via os.ExpandEnv
//  2. leading ~ or ~/ replaced with the user's home directory
//  3. if still relative, joined against baseDir (or CWD if baseDir is empty)
//  4. filepath.Clean
//
// The "~user" form is intentionally not supported.
func Expand(p, baseDir string) (string, error) {
	if p == "" {
		return "", errors.New("empty path")
	}
	p = os.ExpandEnv(p)
	if strings.HasPrefix(p, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		switch {
		case p == "~":
			p = home
		case strings.HasPrefix(p, "~/"):
			p = filepath.Join(home, p[2:])
		}
	}
	if !filepath.IsAbs(p) {
		if baseDir == "" {
			cwd, err := os.Getwd()
			if err != nil {
				return "", err
			}
			baseDir = cwd
		}
		p = filepath.Join(baseDir, p)
	}
	return filepath.Clean(p), nil
}
