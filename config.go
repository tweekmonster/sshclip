package sshclip

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

var MaxStorageMB = 10

var xdgEnv = map[string]string{
	"XDG_DATA_HOME":   "~/.local/share",
	"XDG_CONFIG_HOME": "~/.config",
}

var configRoot = "$XDG_CONFIG_HOME/sshclip"
var dataRoot = "$XDG_DATA_HOME/sshclip"

func init() {
	for k, v := range xdgEnv {
		path := os.Getenv(k)
		if path != "" {
			path = ExpandUser(path)
			if err := EnsureDirectory(path, 0); err != nil {
				log.Fatal("Couldn't ensure directory exists:", err)
			}
			os.Setenv(k, path)
			continue
		}

		path = ExpandUser(v)
		if err := EnsureDirectory(path, 0); err != nil {
			log.Fatal("Couldn't ensure directory exists:", err)
		}

		os.Setenv(k, path)
	}

	configRoot = os.ExpandEnv(configRoot)
	dataRoot = os.ExpandEnv(dataRoot)
}

// OpenConfig opens a file within $XDG_CONFIG_HOME for reading.
func OpenConfig(filename string) (*os.File, error) {
	return os.Open(filepath.Join(configRoot, filename))
}

// DataFilePath returns the full path within $XDG_DATA_HOME.
func DataFilePath(path string) string {
	return filepath.Join(dataRoot, path)
}

// DataFileExists checks if a file exists within $XDG_DATA_HOME.
func DataFileExists(path string) bool {
	if _, err := os.Stat(DataFilePath(path)); os.IsNotExist(err) {
		return false
	}

	return true
}

// OpenDataFile opens a file within $XDG_DATA_HOME.
func OpenDataFile(path string, flag int, perm os.FileMode) (*os.File, error) {
	path = DataFilePath(path)
	dir, _ := filepath.Split(path)

	if err := EnsureDirectory(dir, 0700); err != nil {
		return nil, err
	}

	return os.OpenFile(path, flag, perm)
}

// ReadData reads data from a file within $XDG_DATA_HOME.
func ReadData(path string) ([]byte, error) {
	file, err := OpenDataFile(path, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(file)
}

// WriteData writes data to a file within $XDG_DATA_HOME.
func WriteData(path string, data []byte, perm os.FileMode) error {
	file, err := OpenDataFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, perm)
	if err != nil {
		return err
	}
	_, err = file.Write(data)
	return err
}
