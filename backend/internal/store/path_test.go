package store

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestDefaultPath(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("checks the XDG layout")
	}
	t.Setenv("HOME", "/home/someone")
	t.Setenv("XDG_DATA_HOME", "")
	if got, _ := DefaultPath(); got != "/home/someone/.local/share/switchboard/switchboard.db" {
		t.Errorf("default: %s", got)
	}
	t.Setenv("XDG_DATA_HOME", "/data")
	if got, _ := DefaultPath(); got != filepath.Join("/data", "switchboard", "switchboard.db") {
		t.Errorf("XDG_DATA_HOME: %s", got)
	}
	t.Setenv("XDG_DATA_HOME", "relative/ignored") // the spec says relative paths are invalid
	if got, _ := DefaultPath(); got != "/home/someone/.local/share/switchboard/switchboard.db" {
		t.Errorf("relative XDG_DATA_HOME: %s", got)
	}
}
