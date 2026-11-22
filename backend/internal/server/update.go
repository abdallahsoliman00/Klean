package server

import (
	"context"
	"net/http"
	"runtime"
	"sync"
	"time"

	"klean/backend/internal/config"
	"klean/backend/internal/update"
)

type updateInfo struct {
	Current   string `json:"current"`
	Latest    string `json:"latest"`
	Available bool   `json:"available"`
	Page      string `json:"page"`
}

// updateCache keeps GitHub's last answer for config.UpdateCheckTTL.
type updateCache struct {
	mu   sync.Mutex
	at   time.Time
	info updateInfo
}

// checkUpdate says whether a newer release exists. A failed check is "no":
// being offline shouldn't nag.
func (s *Server) checkUpdate(w http.ResponseWriter, r *http.Request) {
	c := &s.upd
	c.mu.Lock()
	defer c.mu.Unlock()
	if time.Since(c.at) > config.UpdateCheckTTL {
		info := updateInfo{Current: config.Version}
		ctx, cancel := context.WithTimeout(r.Context(), config.UpdateCheckTimeout)
		defer cancel()
		if rel, err := update.Latest(ctx, http.DefaultClient); err == nil {
			info.Latest, info.Page = rel.Tag, rel.Page
			info.Available = update.Newer(config.Version, rel.Tag)
			c.at = time.Now()
		}
		c.info = info
	}
	writeJSON(w, 200, c.info)
}

// applyUpdate downloads the latest release for this platform, checks it and
// puts it in place of this executable. It runs from the next start.
func (s *Server) applyUpdate(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), config.UpdateDownloadTimeout)
	defer cancel()
	rel, err := update.Latest(ctx, http.DefaultClient)
	if err != nil {
		fail(w, http.StatusBadGateway, "Couldn't reach GitHub to download the update.")
		return
	}
	if !update.Newer(config.Version, rel.Tag) {
		fail(w, http.StatusConflict, "Klean is already up to date.")
		return
	}
	manual := " Download it from " + rel.Page
	asset := update.AssetName(rel.Tag, runtime.GOOS, runtime.GOARCH)
	if rel.Assets[asset] == "" || rel.Assets[config.ChecksumsAsset] == "" {
		fail(w, http.StatusNotFound, "The release has no build for this computer."+manual)
		return
	}
	archive, err := update.Download(ctx, http.DefaultClient, rel.Assets[asset])
	if err != nil {
		fail(w, http.StatusBadGateway, "The download failed."+manual)
		return
	}
	sums, err := update.Download(ctx, http.DefaultClient, rel.Assets[config.ChecksumsAsset])
	if err == nil {
		err = update.Verify(archive, string(sums), asset)
	}
	if err != nil {
		fail(w, http.StatusBadGateway, "The download couldn't be verified, so nothing was changed."+manual)
		return
	}
	bin, err := update.Extract(archive, asset)
	if err != nil {
		fail(w, http.StatusBadGateway, "The download has no Klean executable in it."+manual)
		return
	}
	exe, err := update.Executable()
	if err == nil {
		err = update.Replace(exe, bin)
	}
	if err != nil {
		fail(w, http.StatusInternalServerError, "Couldn't replace Klean's executable ("+err.Error()+")."+manual)
		return
	}
	s.upd.mu.Lock()
	s.upd.info.Available = false
	s.upd.mu.Unlock()
	writeJSON(w, 200, map[string]string{"version": rel.Tag})
}
