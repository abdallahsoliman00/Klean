package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestNewer(t *testing.T) {
	cases := []struct {
		cur, latest string
		want        bool
	}{
		{"0.0.1", "v0.0.2", true},
		{"v0.0.2", "v0.0.2", false},
		{"0.1.0", "v0.0.9", false},
		{"0.9.9", "v1.0.0", true},
		{"0.0.10", "v0.0.9", false},
		{"dev", "v9.9.9", false},
		{"0.0.1", "nightly", false},
	}
	for _, c := range cases {
		if got := Newer(c.cur, c.latest); got != c.want {
			t.Errorf("Newer(%q, %q) = %v", c.cur, c.latest, got)
		}
	}
}

func targz(t *testing.T, name string, body []byte) []byte {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	_ = tw.WriteHeader(&tar.Header{Name: "README.md", Mode: 0o644, Size: 2, Typeflag: tar.TypeReg})
	_, _ = tw.Write([]byte("hi"))
	_ = tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(body)), Typeflag: tar.TypeReg})
	_, _ = tw.Write(body)
	if tw.Close() != nil || gz.Close() != nil {
		t.Fatal("writing tar.gz")
	}
	return buf.Bytes()
}

func zipped(t *testing.T, name string, body []byte) []byte {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create(name)
	_, _ = w.Write(body)
	if zw.Close() != nil {
		t.Fatal("writing zip")
	}
	return buf.Bytes()
}

func TestExtractAndVerify(t *testing.T) {
	bin := []byte("new klean")
	for _, goos := range []string{"linux", "windows"} {
		asset := AssetName("v1.2.3", goos, "amd64")
		var archive []byte
		if goos == "windows" {
			archive = zipped(t, "klean_1.2.3_windows_amd64/klean.exe", bin)
		} else {
			archive = targz(t, "klean_1.2.3_linux_amd64/klean", bin)
		}
		got, err := Extract(archive, asset)
		if err != nil || !bytes.Equal(got, bin) {
			t.Errorf("%s: Extract = %q, %v", asset, got, err)
		}
		sum := sha256.Sum256(archive)
		sums := "deadbeef  other.zip\n" + hex.EncodeToString(sum[:]) + "  " + asset + "\n"
		if err := Verify(archive, sums, asset); err != nil {
			t.Errorf("%s: Verify: %v", asset, err)
		}
		if Verify(append(archive, 0), sums, asset) == nil {
			t.Errorf("%s: a changed archive passed", asset)
		}
		if Verify(archive, "deadbeef  other.zip\n", asset) == nil {
			t.Errorf("%s: a missing checksum passed", asset)
		}
	}
	if _, err := Extract(targz(t, "elsewhere/klean", bin), AssetName("1.2.3", "linux", "amd64")); err == nil {
		t.Error("found a binary at the wrong path")
	}
}

func TestReplace(t *testing.T) {
	exe := filepath.Join(t.TempDir(), "klean")
	if err := os.WriteFile(exe, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Replace(exe, []byte("new")); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(exe)
	if string(got) != "new" {
		t.Errorf("exe holds %q", got)
	}
	if _, err := os.Stat(exe + ".old"); !os.IsNotExist(err) {
		t.Error(".old left behind")
	}
}
