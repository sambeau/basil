package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/sambeau/basil/auth"
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
	// Check for subcommands first
	if len(args) > 0 {
		switch args[0] {
		case "users":
			return runUsersCommand(args[1:], stdout, stderr, getenv)
		}
	}

	// Default: run server
	return runServer(ctx, args, stdout, stderr, getenv)
}

// runServer runs the Basil web server
func runServer(ctx context.Context, args []string, stdout, stderr io.Writer, getenv func(string) string) error {
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
  basil users <command> [options]

Server Options:
  --config PATH    Path to config file (default: auto-detect)
  --dev            Development mode (HTTP on localhost)
  --port PORT      Override listen port
  --version        Show version
  --help           Show this help

User Management:
  basil users list            List all users
  basil users show <id>       Show user details
  basil users delete <id>     Delete a user
  basil users reset <id>      Generate new recovery codes

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
  basil users list          List all registered users
  basil users reset usr_abc Generate new recovery codes for user

`)
}

// runUsersCommand handles the `basil users` subcommand.
func runUsersCommand(args []string, stdout, stderr io.Writer, getenv func(string) string) error {
	// Set up flags for users command
	flags := flag.NewFlagSet("basil users", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	var (
		configPath = flags.String("config", "", "Path to config file")
		force      = flags.Bool("force", false, "Skip confirmation prompts")
	)

	if len(args) == 0 {
		printUsersUsage(stderr)
		return fmt.Errorf("missing users subcommand")
	}

	// Get the subcommand
	subCmd := args[0]

	// Parse remaining flags
	if err := flags.Parse(args[1:]); err != nil {
		printUsersUsage(stderr)
		return err
	}

	// Load config to find auth database path
	cfg, configFile, err := config.LoadWithPath(*configPath, getenv)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Check if auth is enabled
	if !cfg.Auth.Enabled {
		return fmt.Errorf("authentication is not enabled in config")
	}

	// Determine auth database path
	dbPath := authDBPath(configFile)

	// Open auth database
	db, err := auth.OpenDB(dbPath)
	if err != nil {
		return fmt.Errorf("opening auth database: %w", err)
	}
	defer db.Close()

	// Execute subcommand
	switch subCmd {
	case "list":
		return usersListCmd(db, stdout)
	case "show":
		if flags.NArg() == 0 {
			return fmt.Errorf("missing user ID")
		}
		return usersShowCmd(db, flags.Arg(0), stdout)
	case "delete":
		if flags.NArg() == 0 {
			return fmt.Errorf("missing user ID")
		}
		return usersDeleteCmd(db, flags.Arg(0), stdout, stderr, *force)
	case "reset":
		if flags.NArg() == 0 {
			return fmt.Errorf("missing user ID")
		}
		return usersResetCmd(db, flags.Arg(0), stdout)
	default:
		printUsersUsage(stderr)
		return fmt.Errorf("unknown users subcommand: %s", subCmd)
	}
}

func printUsersUsage(w io.Writer) {
	fmt.Fprintf(w, `basil users - Manage authentication users

Usage:
  basil users <command> [options] [args]

Commands:
  list              List all users
  show <id>         Show user details
  delete <id>       Delete a user
  reset <id>        Generate new recovery codes

Options:
  --config PATH     Path to config file
  --force           Skip confirmation prompts (for delete)

Examples:
  basil users list
  basil users show usr_abc123
  basil users delete usr_abc123 --force
  basil users reset usr_abc123

`)
}

// authDBPath returns the path to the auth database given a config file path.
func authDBPath(configFile string) string {
	// Auth database is stored alongside the config file as .basil-auth.db
	if configFile == "" {
		return ".basil-auth.db"
	}
	dir := filepath.Dir(configFile)
	return filepath.Join(dir, ".basil-auth.db")
}

// usersListCmd lists all users.
func usersListCmd(db *auth.DB, stdout io.Writer) error {
	users, err := db.ListUsers()
	if err != nil {
		return fmt.Errorf("listing users: %w", err)
	}

	if len(users) == 0 {
		fmt.Fprintln(stdout, "No users found.")
		return nil
	}

	// Print header
	fmt.Fprintf(stdout, "%-20s %-30s %-30s %s\n", "ID", "NAME", "EMAIL", "CREATED")
	fmt.Fprintln(stdout, "-------------------------------------------------------------------------------------")

	// Print users
	for _, u := range users {
		email := u.Email
		if email == "" {
			email = "(none)"
		}
		fmt.Fprintf(stdout, "%-20s %-30s %-30s %s\n",
			u.ID, u.Name, email, u.CreatedAt.Format("2006-01-02 15:04"))
	}

	fmt.Fprintf(stdout, "\nTotal: %d user(s)\n", len(users))
	return nil
}

// usersShowCmd shows details for a single user.
func usersShowCmd(db *auth.DB, userID string, stdout io.Writer) error {
	user, err := db.GetUser(userID)
	if err != nil {
		return fmt.Errorf("getting user: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user not found: %s", userID)
	}

	fmt.Fprintf(stdout, "User ID:    %s\n", user.ID)
	fmt.Fprintf(stdout, "Name:       %s\n", user.Name)
	if user.Email != "" {
		fmt.Fprintf(stdout, "Email:      %s\n", user.Email)
	} else {
		fmt.Fprintln(stdout, "Email:      (none)")
	}
	fmt.Fprintf(stdout, "Created:    %s\n", user.CreatedAt.Format("2006-01-02 15:04:05"))

	// Get credential count
	creds, err := db.GetCredentialsByUser(userID)
	if err == nil {
		fmt.Fprintf(stdout, "Passkeys:   %d\n", len(creds))
	}

	// Get recovery code count
	count, err := db.GetRecoveryCodeCount(userID)
	if err == nil {
		fmt.Fprintf(stdout, "Recovery:   %d code(s) remaining\n", count)
	}

	return nil
}

// usersDeleteCmd deletes a user.
func usersDeleteCmd(db *auth.DB, userID string, stdout, stderr io.Writer, force bool) error {
	// Check user exists first
	user, err := db.GetUser(userID)
	if err != nil {
		return fmt.Errorf("getting user: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user not found: %s", userID)
	}

	// Confirm unless --force
	if !force {
		fmt.Fprintf(stderr, "Delete user %q (%s)? [y/N] ", user.Name, user.ID)
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Fprintln(stdout, "Cancelled.")
			return nil
		}
	}

	// Delete user (cascades to credentials, sessions, recovery codes)
	if err := db.DeleteUser(userID); err != nil {
		return fmt.Errorf("deleting user: %w", err)
	}

	fmt.Fprintf(stdout, "Deleted user: %s\n", user.ID)
	return nil
}

// usersResetCmd generates new recovery codes for a user.
func usersResetCmd(db *auth.DB, userID string, stdout io.Writer) error {
	// Check user exists first
	user, err := db.GetUser(userID)
	if err != nil {
		return fmt.Errorf("getting user: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user not found: %s", userID)
	}

	// Generate new codes (deletes old ones)
	codes, err := db.GenerateRecoveryCodes(userID, 8)
	if err != nil {
		return fmt.Errorf("generating recovery codes: %w", err)
	}

	fmt.Fprintf(stdout, "New recovery codes for %s (%s):\n\n", user.Name, user.ID)
	for _, code := range codes {
		fmt.Fprintf(stdout, "  %s\n", code)
	}
	fmt.Fprintln(stdout, "\nSave these codes securely. They cannot be shown again.")

	return nil
}
