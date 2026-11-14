// Package config holds Klean's compiled-in defaults and limits in one place.
//
// There is no config file: Klean stays a single executable. The few settings
// meant to change at run time are flags and environment variables (see
// backend/main.go), which take their defaults from here; everything else is
// tuned by editing these constants.
package config

import "time"

// Startup and the HTTP server.
const (
	DefaultHost        = "127.0.0.1" // --host / KLEAN_HOST
	DefaultPort        = 8787        // --port / KLEAN_PORT
	DefaultCORSOrigins = "*"         // KLEAN_CORS, comma-separated origins

	// AppID is reported by /api/health, so a second copy started on the same
	// port recognises the one already running.
	AppID = "klean"

	ReadHeaderTimeout  = 10 * time.Second // slow-loris guard on incoming requests
	ShutdownTimeout    = 3 * time.Second  // grace period for open requests on Ctrl+C
	HealthCheckTimeout = 2 * time.Second  // asking a running copy whether it is Klean

	// FakeProviderPort is where `go run ./backend/cmd/fakeprovider` listens
	// when no port is given.
	FakeProviderPort = "9100"
)

// Limits on what the API accepts.
const (
	MaxRequestBody = 10 << 20 // bytes in a JSON request body

	ProviderNameMaxChars = 80
	ChatTitleMaxChars    = 200
	PresetNameMaxChars   = 80

	MinTemperature = 0.0
	MaxTemperature = 2.0

	MinContextWindow = 512
	MaxContextWindow = 2_000_000
)
