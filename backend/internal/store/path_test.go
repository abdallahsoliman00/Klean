package store

import (
	"path/filepath"
	"runtime"
	"testing"

	"klean/backend/internal/config"
)

func TestDefaultPath(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("checks the XDG layout")
	}
	home := filepath.Join("/home/someone", ".local", "share", config.DataDirName, config.DBFileName)
	t.Setenv("HOME", "/home/someone")
	t.Setenv("XDG_DATA_HOME", "")
	if got, _ := DefaultPath(); got != home {
		t.Errorf("default: %s", got)
	}
	t.Setenv("XDG_DATA_HOME", "/data")
	if got, _ := DefaultPath(); got != filepath.Join("/data", config.DataDirName, config.DBFileName) {
		t.Errorf("XDG_DATA_HOME: %s", got)
	}
	t.Setenv("XDG_DATA_HOME", "relative/ignored") // the spec says relative paths are invalid
	if got, _ := DefaultPath(); got != home {
		t.Errorf("relative XDG_DATA_HOME: %s", got)
	}
}
