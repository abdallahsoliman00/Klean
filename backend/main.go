// Switchboard backend. Run with:  go run .  (from the backend folder)
//
// Settings (flags or environment variables):
//
//	--host / SWITCHBOARD_HOST          default 127.0.0.1
//	--port / SWITCHBOARD_PORT          default 8787
//	--db   / SWITCHBOARD_DB            default ./data/switchboard.db
//	--frontend / SWITCHBOARD_FRONTEND  default ../frontend ("" to serve the API only)
//	SWITCHBOARD_CORS                   default * (comma-separated origins)
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"switchboard/internal/server"
	"switchboard/internal/store"
)

func env(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

// baseDir is the backend folder: the working directory under `go run`,
// otherwise the folder holding the executable.
func baseDir() string {
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		if _, err := os.Stat(filepath.Join(dir, "..", "frontend")); err == nil {
			return dir
		}
	}
	wd, _ := os.Getwd()
	return wd
}

func main() {
	here := baseDir()
	port, err := strconv.Atoi(env("SWITCHBOARD_PORT", "8787"))
	if err != nil {
		log.Fatalf("SWITCHBOARD_PORT must be a number")
	}
	host := flag.String("host", env("SWITCHBOARD_HOST", "127.0.0.1"), "address to listen on")
	flag.IntVar(&port, "port", port, "port to listen on")
	dbPath := flag.String("db", env("SWITCHBOARD_DB", filepath.Join(here, "data", "switchboard.db")), "SQLite database file")
	frontend := flag.String("frontend", env("SWITCHBOARD_FRONTEND", filepath.Join(here, "..", "frontend")), `frontend folder ("" to serve the API only)`)
	flag.Parse()

	db, err := store.Open(*dbPath)
	if err != nil {
		log.Fatalf("opening database %s: %v", *dbPath, err)
	}
	defer db.Close()

	srv := &http.Server{
		Addr:              net.JoinHostPort(*host, strconv.Itoa(port)),
		Handler:           server.New(db, *frontend, env("SWITCHBOARD_CORS", "*")),
		ReadHeaderTimeout: 10 * time.Second,
	}
	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		log.Fatalf("%v", err)
	}
	shown := *host
	if shown == "127.0.0.1" || shown == "0.0.0.0" || shown == "" {
		shown = "localhost"
	}
	fmt.Printf("\n  Switchboard is running at http://%s:%d\n  Database: %s\n\n", shown, port, *dbPath)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdown)
	}()
	if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("%v", err)
	}
}
