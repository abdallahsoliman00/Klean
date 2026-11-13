package store

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"

	"klean/backend/internal/config"
)

// DefaultPath is where the database lives unless --db or KLEAN_DB says
// otherwise. It's per user and independent of where the executable is, so the
// binary can be moved or replaced without losing chats:
//
//	Linux and others  $XDG_DATA_HOME/klean (~/.local/share/klean)
//	macOS             ~/Library/Application Support/Klean
//	Windows           %LOCALAPPDATA%\Klean
//
// It is deliberately not a cache folder: those may be cleared at any time.
func DefaultPath() (string, error) {
	var dir string
	switch runtime.GOOS {
	case "windows":
		dir = os.Getenv("LOCALAPPDATA")
		if dir == "" {
			return "", errors.New("%LOCALAPPDATA% is not set; pass --db")
		}
		dir = filepath.Join(dir, config.DataDirNameDesktop)
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, "Library", "Application Support", config.DataDirNameDesktop)
	default:
		dir = os.Getenv("XDG_DATA_HOME")
		if dir == "" || !filepath.IsAbs(dir) {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			dir = filepath.Join(home, ".local", "share")
		}
		dir = filepath.Join(dir, config.DataDirName)
	}
	return filepath.Join(dir, config.DBFileName), nil
}
