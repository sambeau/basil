package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/sambeau/basil/config"
	"github.com/sambeau/basil/server"
)

// Version information, set at build time via -ldflags
var (
	Version = "dev"     // -X main.Version=$(git describe --tags --always)
	Commit  = "unknown" // -X main.Commit=$(git rev-parse --short HEAD)
)

func main() {
	ctx := context.Background()
	if err := run(ctx, os.Args[1:], os.Stdout, os.Stderr, os.Getenv); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// run is the main entry point, designed for testability (Mat Ryer pattern)
func run(ctx context.Context, args []string, stdout, stderr io.Writer, getenv func(string) string) error {
	// Set up flags
	flags := flag.NewFlagSet("basil", flag.ContinueOnError)
	flags.SetOutput(io.Discard) // Suppress default -h output

	var (
		configPath  = flags.String("config", "", "Path to config file")
		devMode     = flags.Bool("dev", false, "Development mode (HTTP on localhost)")
		port        = flags.Int("port", 0, "Override listen port")
		showVersion = flags.Bool("version", false, "Show version")
		showHelp    = flags.Bool("help", false, "Show help")
	)

	// Parse flags
	if err := flags.Parse(args); err != nil {
		// Handle -h/--help: flag package returns ErrHelp
		if errors.Is(err, flag.ErrHelp) {
			printUsage(stdout)
			return nil
		}
		// For other errors, show usage then error
		printUsage(stderr)
		return err
	}

	// Handle explicit --help flag
	if *showHelp {
		printUsage(stdout)
		return nil
	}

	// Handle --version
	if *showVersion {
		fmt.Fprintf(stdout, "basil version %s (%s)\n", Version, Commit)
		return nil
	}

	// Set up signal handling for graceful shutdown
	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Load configuration
	cfg, configFile, err := config.LoadWithPath(*configPath, getenv)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Apply CLI overrides
	if *devMode {
		cfg.Server.Dev = true
	}
	if *port != 0 {
		cfg.Server.Port = *port
	}

	// Full validation after CLI overrides applied
	if err := config.Validate(cfg); err != nil {
		return fmt.Errorf("config validation: %w", err)
	}

	// Create server
	srv, err := server.New(cfg, configFile, stdout, stderr)
	if err != nil {
		return fmt.Errorf("creating server: %w", err)
	}

	// Set up SIGHUP handler for script cache reload (production hot reload)
	sighup := make(chan os.Signal, 1)
	signal.Notify(sighup, syscall.SIGHUP)
	go func() {
		for range sighup {
			fmt.Fprintf(stdout, "Received SIGHUP - reloading scripts...\n")
			srv.ReloadScripts()
		}
	}()

	return srv.Run(ctx)
}

func printUsage(w io.Writer) {
	fmt.Fprintf(w, `basil - A web server for Parsley

Usage:
  basil [options]

Options:
  --config PATH    Path to config file (default: auto-detect)
  --dev            Development mode (HTTP on localhost)
  --port PORT      Override listen port
  --version        Show version
  --help           Show this help

Config Resolution:
  1. --config flag
  2. BASIL_CONFIG environment variable
  3. ./basil.yaml
  4. ~/.config/basil/basil.yaml

Signals:
  SIGHUP           Reload scripts (clear cache, re-parse on next request)
  SIGINT/SIGTERM   Graceful shutdown

Examples:
  basil                     Start with auto-detected config
  basil --dev               Development mode (HTTP on localhost:8080)
  basil --config app.yaml   Use specific config file
  basil --dev --port 3000   Dev mode on port 3000
  kill -HUP <pid>           Reload scripts without restart

`)
}
