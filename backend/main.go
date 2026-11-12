// Klean: a local chat UI for OpenAI-compatible servers, in one executable.
//
// Build from the repository root:  go build -o klean ./backend
//
// Settings (flags or environment variables):
//
//	--host / KLEAN_HOST          default 127.0.0.1
//	--port / KLEAN_PORT          default 8787
//	--db   / KLEAN_DB            default: per-user data folder (see store.DefaultPath)
//	--frontend / KLEAN_FRONTEND  default: the UI built into the executable;
//	                                   a folder serves that instead, "none" serves only the API
//	--no-browser / KLEAN_NO_BROWSER=1  don't open the browser on start
//	--headless                         no browser and no terminal window (services, autostart)
//	KLEAN_CORS                   default * (comma-separated origins)
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"klean/backend/internal/launch"
	"klean/backend/internal/server"
	"klean/backend/internal/store"
	"klean/frontend"
)

func env(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

func main() {
	// Double-clicked in a file manager: run inside a terminal window instead,
	// so closing that window stops the server.
	if !hasFlag("headless") && launch.NeedsTerminal() && launch.OpenInTerminal() {
		return
	}
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "\n  %v\n", err)
		launch.PauseIfOwnWindow()
		os.Exit(1)
	}
}

func run() error {
	port, err := strconv.Atoi(env("KLEAN_PORT", "8787"))
	if err != nil {
		return errors.New("KLEAN_PORT must be a number")
	}
	defaultDB := os.Getenv("KLEAN_DB")
	if defaultDB == "" {
		if defaultDB, err = store.DefaultPath(); err != nil {
			return err
		}
	}
	host := flag.String("host", env("KLEAN_HOST", "127.0.0.1"), "address to listen on")
	flag.IntVar(&port, "port", port, "port to listen on")
	dbPath := flag.String("db", defaultDB, "SQLite database file")
	frontendDir := flag.String("frontend", env("KLEAN_FRONTEND", ""),
		`serve the UI from this folder instead of the built-in copy ("none" = API only)`)
	noBrowser := flag.Bool("no-browser", env("KLEAN_NO_BROWSER", "") != "", "don't open the browser on start")
	headless := flag.Bool("headless", false, "don't open a browser or a terminal window (for services and autostart)")
	flag.Parse()
	*noBrowser = *noBrowser || *headless

	var static fs.FS = frontend.Files
	switch *frontendDir {
	case "":
	case "none":
		static = nil
	default:
		if st, err := os.Stat(*frontendDir); err != nil || !st.IsDir() {
			return fmt.Errorf("--frontend %s is not a folder", *frontendDir)
		}
		static = os.DirFS(*frontendDir)
	}

	shown := *host
	if shown == "127.0.0.1" || shown == "0.0.0.0" || shown == "" || shown == "::" {
		shown = "localhost"
	}
	url := fmt.Sprintf("http://%s/", net.JoinHostPort(shown, strconv.Itoa(port)))
	addr := net.JoinHostPort(*host, strconv.Itoa(port))

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		// (The errno for "address in use" differs per OS, so ask the port instead.)
		if !isKlean(url) {
			return fmt.Errorf("can't listen on port %d (%v).\n  If another program uses it, start with --port to pick another one", port, err)
		}
		// Already running (perhaps in another window): just show it.
		fmt.Printf("\n  Klean is already running at %s\n\n", url)
		if static != nil && !*noBrowser {
			_ = launch.OpenBrowser(url)
		}
		return nil
	}

	db, err := store.Open(*dbPath)
	if err != nil {
		ln.Close()
		return fmt.Errorf("opening database %s: %w", *dbPath, err)
	}
	defer db.Close()

	srv := &http.Server{
		Handler:           server.New(db, static, env("KLEAN_CORS", "*")),
		ReadHeaderTimeout: 10 * time.Second,
	}
	fmt.Printf("\n  Klean is running at %s\n  Database: %s\n", url, *dbPath)
	if launch.OwnsWindow() {
		fmt.Print("\n  Closing this window stops Klean.\n\n")
	} else {
		fmt.Print("\n  Press Ctrl+C to stop.\n\n")
	}
	if static != nil && !*noBrowser {
		// The listener is already open, so the page can't load before the server is up.
		if err := launch.OpenBrowser(url); err != nil {
			fmt.Printf("  Couldn't open a browser (%v). Open %s yourself.\n\n", err, url)
		}
	}

	// SIGHUP arrives when the terminal window is closed.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdown)
	}()
	if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// hasFlag reports whether a boolean flag was given, before flags are parsed.
func hasFlag(name string) bool {
	for _, a := range os.Args[1:] {
		if a == "-"+name || a == "--"+name || a == "--"+name+"=true" || a == "-"+name+"=true" {
			return true
		}
	}
	return false
}

// isKlean reports whether a Klean server answers at url.
func isKlean(url string) bool {
	client := http.Client{Timeout: 2 * time.Second}
	res, err := client.Get(url + "api/health")
	if err != nil {
		return false
	}
	defer res.Body.Close()
	var body struct{ App string }
	return json.NewDecoder(res.Body).Decode(&body) == nil && body.App == "klean"
}
