package config

import "time"

// Version is the release this executable was built from, set by the release
// workflow with -ldflags "-X klean/backend/internal/config.Version=1.2.3".
// Builds without it are "dev" and never offer an update.
var Version = "dev"

// Updating Klean from its GitHub releases.
const (
	ReleasesLatestURL = "https://api.github.com/repos/abdallahsoliman00/Klean/releases/latest"
	// ChecksumsAsset is the release file holding every archive's sha256.
	ChecksumsAsset = "checksums.txt"

	// UpdateCheckTTL is how long one answer from GitHub is reused; unauthenticated
	// clients get 60 requests an hour.
	UpdateCheckTTL        = 6 * time.Hour
	UpdateCheckTimeout    = 10 * time.Second
	UpdateDownloadTimeout = 5 * time.Minute
	UpdateMaxBytes        = 200 << 20 // the largest archive or binary accepted
)
