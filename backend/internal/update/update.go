// Package update replaces this executable with the latest GitHub release.
package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"klean/backend/internal/config"
)

// Release is the latest published release.
type Release struct {
	Tag    string            // e.g. "v1.2.3"
	Page   string            // the release's web page
	Assets map[string]string // file name -> download URL
}

// Newer reports whether latest is a higher vX.Y.Z than current. Anything that
// isn't a version (a "dev" build) is never older.
func Newer(current, latest string) bool {
	c, ok1 := parse(current)
	l, ok2 := parse(latest)
	if !ok1 || !ok2 {
		return false
	}
	for i := range c {
		if l[i] != c[i] {
			return l[i] > c[i]
		}
	}
	return false
}

func parse(v string) ([3]int, bool) {
	var out [3]int
	parts := strings.Split(strings.TrimPrefix(v, "v"), ".")
	if len(parts) != len(out) {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return out, false
		}
		out[i] = n
	}
	return out, true
}

// AssetName is the archive name release.yml gives a target, without the "v".
func AssetName(version, goos, goarch string) string {
	ext := "tar.gz"
	if goos == "windows" {
		ext = "zip"
	}
	return fmt.Sprintf("klean_%s_%s_%s.%s", strings.TrimPrefix(version, "v"), goos, goarch, ext)
}

// binaryPath is where the executable sits inside an archive.
func binaryPath(asset string) string {
	dir := strings.TrimSuffix(strings.TrimSuffix(asset, ".tar.gz"), ".zip")
	if strings.HasSuffix(asset, ".zip") {
		return dir + "/klean.exe"
	}
	return dir + "/klean"
}

// Latest asks GitHub for the latest release.
func Latest(ctx context.Context, client *http.Client) (Release, error) {
	var body struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
		Assets  []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		} `json:"assets"`
	}
	data, err := Download(ctx, client, config.ReleasesLatestURL)
	if err != nil {
		return Release{}, err
	}
	if err := json.Unmarshal(data, &body); err != nil {
		return Release{}, err
	}
	r := Release{Tag: body.TagName, Page: body.HTMLURL, Assets: map[string]string{}}
	for _, a := range body.Assets {
		r.Assets[a.Name] = a.URL
	}
	return r, nil
}

// Download fetches url, refusing anything over config.UpdateMaxBytes.
func Download(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s answered %s", url, res.Status)
	}
	data, err := io.ReadAll(io.LimitReader(res.Body, config.UpdateMaxBytes+1))
	if err == nil && len(data) > config.UpdateMaxBytes {
		err = errors.New("download is larger than expected")
	}
	return data, err
}

// Verify checks archive against its line in checksums.txt ("<hex>  <name>").
func Verify(archive []byte, checksums, name string) error {
	sum := sha256.Sum256(archive)
	for _, line := range strings.Split(checksums, "\n") {
		f := strings.Fields(line)
		if len(f) == 2 && strings.TrimPrefix(f[1], "*") == name {
			if f[0] != hex.EncodeToString(sum[:]) {
				return fmt.Errorf("%s doesn't match its checksum", name)
			}
			return nil
		}
	}
	return fmt.Errorf("%s isn't in %s", name, config.ChecksumsAsset)
}

// Extract returns the executable inside the archive named asset.
func Extract(archive []byte, asset string) ([]byte, error) {
	want := binaryPath(asset)
	if strings.HasSuffix(asset, ".zip") {
		zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
		if err != nil {
			return nil, err
		}
		for _, f := range zr.File {
			if path.Clean(f.Name) == want {
				rc, err := f.Open()
				if err != nil {
					return nil, err
				}
				defer rc.Close()
				return io.ReadAll(io.LimitReader(rc, config.UpdateMaxBytes))
			}
		}
	} else {
		gz, err := gzip.NewReader(bytes.NewReader(archive))
		if err != nil {
			return nil, err
		}
		tr := tar.NewReader(gz)
		for {
			h, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, err
			}
			if path.Clean(h.Name) == want && h.Typeflag == tar.TypeReg {
				return io.ReadAll(io.LimitReader(tr, config.UpdateMaxBytes))
			}
		}
	}
	return nil, fmt.Errorf("%s has no %s", asset, want)
}

// Replace puts newBin where exe is. The running file is moved aside first,
// which Windows allows for a running .exe (deleting it waits for Cleanup).
func Replace(exe string, newBin []byte) error {
	tmp, old := exe+".new", exe+".old"
	if err := os.WriteFile(tmp, newBin, 0o755); err != nil {
		return err
	}
	_ = os.Remove(old)
	if err := os.Rename(exe, old); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, exe); err != nil {
		_ = os.Rename(old, exe) // put the working one back
		os.Remove(tmp)
		return err
	}
	_ = os.Remove(old)
	return nil
}

// Executable is the path of this program's file, through any symlink.
func Executable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(exe)
}

// Cleanup removes what Replace couldn't delete while the old copy ran.
func Cleanup(exe string) { _ = os.Remove(exe + ".old") }
