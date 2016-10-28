package sshclip

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/tweekmonster/luser"
)

// ExpandUser expands the ~/ portion of the path with the user's home
// directory.
func ExpandUser(path string) string {
	if strings.HasPrefix(path, "~/") {
		if u, err := luser.Current(); err == nil {
			return filepath.Join(u.HomeDir, strings.TrimPrefix(path, "~/"))
		}
	}

	return path
}

// EnsureDirectory ensures that a directory exists and its permissions are
// correct.
func EnsureDirectory(path string, perm os.FileMode) error {
	path, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	st, err := os.Stat(path)
	if err == nil {
		if st.IsDir() {
			if perm != 0 && st.Mode().Perm()^perm.Perm() != 0 {
				return os.Chmod(path, perm.Perm())
			}
			return nil
		}

		return os.ErrExist
	}

	if !os.IsNotExist(err) {
		return err
	}

	return os.MkdirAll(path, 0700)
}
