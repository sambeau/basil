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
	"strings"
	"syscall"

	"github.com/sambeau/basil/server"
	"github.com/sambeau/basil/server/auth"
	"github.com/sambeau/basil/server/config"
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
		case "apikey":
			return runAPIKeyCommand(args[1:], stdout, stderr, getenv)
		case "auth":
			return runAuthCommand(args[1:], stdout, stderr, getenv)
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
		quietMode   = flags.Bool("quiet", false, "Suppress request logs (sets log level to error)")
		port        = flags.Int("port", 0, "Override listen port")
		profile     = flags.String("profile", "", "Developer profile to apply")
		initFolder  = flags.String("init", "", "Create a new Basil project in the specified folder")
		showVersion = flags.Bool("version", false, "Show version")
		showHelp    = flags.Bool("help", false, "Show help")
	)
	// -as alias for --profile
	flags.StringVar(profile, "as", "", "Alias for --profile")

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

	// Handle --init
	if *initFolder != "" {
		return runInitCommand(*initFolder, stdout, stderr)
	}

	// Set up signal handling for graceful shutdown
	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Load configuration
	cfg, configFile, err := config.LoadWithPath(*configPath, getenv)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Apply developer profile if specified
	if *profile != "" {
		if err := config.ApplyDeveloper(cfg, *profile); err != nil {
			return fmt.Errorf("applying profile %q: %w", *profile, err)
		}
	}

	// Apply CLI overrides
	if *devMode {
		cfg.Server.Dev = true
	}
	if *quietMode || cfg.Logging.Quiet {
		cfg.Logging.Level = "error"
	}
	if *port != 0 {
		cfg.Server.Port = *port
	}

	// Full validation after CLI overrides applied
	if err := config.Validate(cfg); err != nil {
		return fmt.Errorf("config validation: %w", err)
	}

	// Show warnings for potential misconfigurations
	for _, warning := range config.Warnings(cfg) {
		fmt.Fprintf(stderr, "warning: %s\n", warning)
	}

	// Build version string
	version := fmt.Sprintf("version %s (%s)", Version, Commit)

	// Create server
	srv, err := server.New(cfg, configFile, version, Commit, stdout, stderr)
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
  basil apikey <command> [options]

Server Options:
  --config PATH      Path to config file (default: auto-detect)
  --dev              Development mode (HTTP on localhost)
  --quiet            Suppress request logs (sets log level to error)
  --port PORT        Override listen port
  --profile NAME     Apply a developer profile from config
  -as NAME           Alias for --profile
  --init FOLDER      Create a new Basil project in the specified folder
  --version          Show version
  --help             Show this help

User Management:
  basil users create           Create a new user
  basil users list             List all users
  basil users show <id>        Show user details
  basil users update <id>      Update user name/email
  basil users set-role <id>    Change user role
  basil users delete <id>      Delete a user
  basil users reset <id>       Generate new recovery codes

API Key Management:
  basil apikey create          Create an API key for a user
  basil apikey list            List API keys for a user
  basil apikey revoke <id>     Revoke an API key

Config Resolution:
  1. --config flag
  2. BASIL_CONFIG environment variable
  3. ./basil.yaml
  4. ~/.config/basil/basil.yaml

Signals:
  SIGHUP           Reload scripts (clear cache, re-parse on next request)
  SIGINT/SIGTERM   Graceful shutdown

Examples:
  basil                       Start with auto-detected config
  basil --dev                 Development mode (HTTP on localhost:8080)
  basil --config app.yaml     Use specific config file
  basil --dev --port 3000     Dev mode on port 3000
  basil --dev -as sam         Dev mode with Sam's config overrides
  basil users create --name "Admin" --email admin@example.com --role admin
  basil users list            List all registered users
  basil apikey create --user usr_abc123 --name "MacBook Git"

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
		name       = flags.String("name", "", "User name")
		email      = flags.String("email", "", "User email")
		role       = flags.String("role", "", "User role (admin/editor)")
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

	// For 'create' command, create the database if it doesn't exist
	// For other commands, require the database to exist
	var db *auth.DB
	if subCmd == "create" {
		db, err = auth.OpenOrCreateDB(dbPath)
	} else {
		db, err = auth.OpenDB(dbPath)
	}
	if err != nil {
		return fmt.Errorf("opening auth database: %w", err)
	}
	defer db.Close()

	// Execute subcommand
	switch subCmd {
	case "create":
		return usersCreateCmd(db, *name, *email, *role, stdout, stderr)
	case "list":
		return usersListCmd(db, stdout)
	case "show":
		if flags.NArg() == 0 {
			return fmt.Errorf("missing user ID")
		}
		return usersShowCmd(db, flags.Arg(0), stdout)
	case "update":
		if flags.NArg() == 0 {
			return fmt.Errorf("missing user ID")
		}
		return usersUpdateCmd(db, flags.Arg(0), *name, *email, stdout)
	case "set-role":
		if flags.NArg() == 0 {
			return fmt.Errorf("missing user ID")
		}
		if flags.NArg() < 2 && *role == "" {
			return fmt.Errorf("missing role (use: admin or editor)")
		}
		targetRole := *role
		if targetRole == "" {
			targetRole = flags.Arg(1)
		}
		return usersSetRoleCmd(db, flags.Arg(0), targetRole, stdout)
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
  create            Create a new user
  list              List all users
  show <id>         Show user details
  update <id>       Update user name/email
  set-role <id>     Change user role
  delete <id>       Delete a user
  reset <id>        Generate new recovery codes

Options:
  --config PATH     Path to config file
  --name NAME       User name (for create/update)
  --email EMAIL     User email (for create/update)
  --role ROLE       User role: admin or editor (for create/set-role)
  --force           Skip confirmation prompts (for delete)

Examples:
  basil users create --name "Admin" --email admin@example.com --role admin
  basil users create --name "Editor" --email editor@example.com
  basil users list
  basil users show usr_abc123
  basil users update usr_abc123 --name "New Name"
  basil users set-role usr_abc123 editor
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

// usersCreateCmd creates a new user.
func usersCreateCmd(db *auth.DB, name, email, role string, stdout, stderr io.Writer) error {
	if name == "" {
		return fmt.Errorf("--name is required")
	}

	// Check if this is the first user
	count, err := db.UserCount()
	if err != nil {
		return fmt.Errorf("checking user count: %w", err)
	}

	// First user is always admin
	if count == 0 {
		if role != "" && role != auth.RoleAdmin {
			fmt.Fprintf(stderr, "Note: First user is always admin (ignoring --role %s)\n", role)
		}
		role = auth.RoleAdmin
	} else if role == "" {
		role = auth.RoleEditor
	}

	// Validate role
	if role != auth.RoleAdmin && role != auth.RoleEditor {
		return fmt.Errorf("invalid role: %s (use: admin or editor)", role)
	}

	user, err := db.CreateUserWithRole(name, email, role)
	if err != nil {
		return fmt.Errorf("creating user: %w", err)
	}

	fmt.Fprintf(stdout, "✓ Created user %s\n", user.ID)
	return nil
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
	fmt.Fprintf(stdout, "%-36s %-20s %-30s %-8s %s\n", "ID", "NAME", "EMAIL", "ROLE", "CREATED")
	fmt.Fprintln(stdout, strings.Repeat("-", 110))

	// Print users
	for _, u := range users {
		email := u.Email
		if email == "" {
			email = "(none)"
		}
		name := u.Name
		if len(name) > 20 {
			name = name[:17] + "..."
		}
		fmt.Fprintf(stdout, "%-36s %-20s %-30s %-8s %s\n",
			u.ID, name, email, u.Role, u.CreatedAt.Format("2006-01-02 15:04"))
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
	fmt.Fprintf(stdout, "Role:       %s\n", user.Role)
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

	// Get API key count
	apiKeys, err := db.GetAPIKeys(userID)
	if err == nil {
		fmt.Fprintf(stdout, "API Keys:   %d\n", len(apiKeys))
	}

	return nil
}

// usersUpdateCmd updates a user's name and/or email.
func usersUpdateCmd(db *auth.DB, userID, name, email string, stdout io.Writer) error {
	if name == "" && email == "" {
		return fmt.Errorf("at least one of --name or --email must be provided")
	}

	// Check user exists
	user, err := db.GetUser(userID)
	if err != nil {
		return fmt.Errorf("getting user: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user not found: %s", userID)
	}

	if err := db.UpdateUser(userID, name, email); err != nil {
		return fmt.Errorf("updating user: %w", err)
	}

	fmt.Fprintf(stdout, "✓ Updated user %s\n", userID)
	return nil
}

// usersSetRoleCmd changes a user's role.
func usersSetRoleCmd(db *auth.DB, userID, role string, stdout io.Writer) error {
	// Check user exists
	user, err := db.GetUser(userID)
	if err != nil {
		return fmt.Errorf("getting user: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user not found: %s", userID)
	}

	// Validate role
	if role != auth.RoleAdmin && role != auth.RoleEditor {
		return fmt.Errorf("invalid role: %s (use: admin or editor)", role)
	}

	// Prevent removing the last admin
	if user.Role == auth.RoleAdmin && role != auth.RoleAdmin {
		adminCount, err := db.CountAdmins()
		if err != nil {
			return fmt.Errorf("checking admin count: %w", err)
		}
		if adminCount <= 1 {
			return fmt.Errorf("cannot remove the last admin user")
		}
	}

	if err := db.SetUserRole(userID, role); err != nil {
		return fmt.Errorf("setting role: %w", err)
	}

	fmt.Fprintf(stdout, "✓ Set role for %s to %s\n", user.Name, role)
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

	// Prevent deleting the last admin
	if user.Role == auth.RoleAdmin {
		adminCount, err := db.CountAdmins()
		if err != nil {
			return fmt.Errorf("checking admin count: %w", err)
		}
		if adminCount <= 1 {
			return fmt.Errorf("cannot delete the last admin user")
		}
	}

	// Confirm unless --force
	if !force {
		fmt.Fprintf(stderr, "⚠ This will delete user %s and all their credentials.\n  Continue? [y/N] ", user.Name)
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Fprintln(stdout, "Cancelled.")
			return nil
		}
	}

	// Delete user (cascades to credentials, sessions, recovery codes, API keys)
	if err := db.DeleteUser(userID); err != nil {
		return fmt.Errorf("deleting user: %w", err)
	}

	fmt.Fprintf(stdout, "✓ Deleted user %s\n", user.ID)
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

// --- API Key Commands ---

// runAPIKeyCommand handles the `basil apikey` subcommand.
func runAPIKeyCommand(args []string, stdout, stderr io.Writer, getenv func(string) string) error {
	flags := flag.NewFlagSet("basil apikey", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	var (
		configPath = flags.String("config", "", "Path to config file")
		userID     = flags.String("user", "", "User ID")
		name       = flags.String("name", "", "API key name/label")
	)

	if len(args) == 0 {
		printAPIKeyUsage(stderr)
		return fmt.Errorf("missing apikey subcommand")
	}

	subCmd := args[0]

	if err := flags.Parse(args[1:]); err != nil {
		printAPIKeyUsage(stderr)
		return err
	}

	// Load config
	cfg, configFile, err := config.LoadWithPath(*configPath, getenv)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if !cfg.Auth.Enabled {
		return fmt.Errorf("authentication is not enabled in config")
	}

	dbPath := authDBPath(configFile)
	db, err := auth.OpenDB(dbPath)
	if err != nil {
		return fmt.Errorf("opening auth database: %w", err)
	}
	defer db.Close()

	switch subCmd {
	case "create":
		if *userID == "" {
			return fmt.Errorf("--user is required")
		}
		if *name == "" {
			return fmt.Errorf("--name is required")
		}
		return apiKeyCreateCmd(db, *userID, *name, stdout)
	case "list":
		if *userID == "" {
			return fmt.Errorf("--user is required")
		}
		return apiKeyListCmd(db, *userID, stdout)
	case "revoke":
		if flags.NArg() == 0 {
			return fmt.Errorf("missing API key ID")
		}
		return apiKeyRevokeCmd(db, flags.Arg(0), stdout)
	default:
		printAPIKeyUsage(stderr)
		return fmt.Errorf("unknown apikey subcommand: %s", subCmd)
	}
}

func printAPIKeyUsage(w io.Writer) {
	fmt.Fprintf(w, `basil apikey - Manage API keys

Usage:
  basil apikey <command> [options] [args]

Commands:
  create            Create a new API key
  list              List API keys for a user
  revoke <id>       Revoke an API key

Options:
  --config PATH     Path to config file
  --user ID         User ID (required for create/list)
  --name NAME       API key name/label (required for create)

Examples:
  basil apikey create --user usr_abc123 --name "MacBook Git"
  basil apikey list --user usr_abc123
  basil apikey revoke key_xyz789

`)
}

// apiKeyCreateCmd creates an API key for a user.
func apiKeyCreateCmd(db *auth.DB, userID, name string, stdout io.Writer) error {
	key, plaintext, err := db.CreateAPIKey(userID, name)
	if err != nil {
		return fmt.Errorf("creating API key: %w", err)
	}

	fmt.Fprintf(stdout, "✓ Created API key: %s\n", plaintext)
	fmt.Fprintf(stdout, "  Key ID: %s\n", key.ID)
	fmt.Fprintln(stdout, "  (save this now — it won't be shown again)")
	return nil
}

// apiKeyListCmd lists API keys for a user.
func apiKeyListCmd(db *auth.DB, userID string, stdout io.Writer) error {
	// Verify user exists
	user, err := db.GetUser(userID)
	if err != nil {
		return fmt.Errorf("getting user: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user not found: %s", userID)
	}

	keys, err := db.GetAPIKeys(userID)
	if err != nil {
		return fmt.Errorf("listing API keys: %w", err)
	}

	if len(keys) == 0 {
		fmt.Fprintln(stdout, "No API keys found.")
		return nil
	}

	fmt.Fprintf(stdout, "%-36s %-20s %-20s %-12s %s\n", "ID", "NAME", "PREFIX", "CREATED", "LAST USED")
	fmt.Fprintln(stdout, strings.Repeat("-", 110))

	for _, k := range keys {
		lastUsed := "never"
		if k.LastUsedAt != nil {
			lastUsed = k.LastUsedAt.Format("2006-01-02")
		}
		name := k.Name
		if len(name) > 20 {
			name = name[:17] + "..."
		}
		fmt.Fprintf(stdout, "%-36s %-20s %-20s %-12s %s\n",
			k.ID, name, k.KeyPrefix, k.CreatedAt.Format("2006-01-02"), lastUsed)
	}

	return nil
}

// apiKeyRevokeCmd revokes an API key.
func apiKeyRevokeCmd(db *auth.DB, keyID string, stdout io.Writer) error {
	// Get key first to show name in confirmation
	key, err := db.GetAPIKey(keyID)
	if err != nil {
		return fmt.Errorf("getting API key: %w", err)
	}
	if key == nil {
		return fmt.Errorf("API key not found: %s", keyID)
	}

	if err := db.DeleteAPIKey(keyID); err != nil {
		return fmt.Errorf("revoking API key: %w", err)
	}

	fmt.Fprintf(stdout, "✓ Revoked API key %q\n", key.Name)
	return nil
}

// runAuthCommand handles auth-related subcommands
func runAuthCommand(args []string, stdout, stderr io.Writer, getenv func(string) string) error {
	flags := flag.NewFlagSet("basil auth", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	var (
		configPath = flags.String("config", "", "Path to config file")
		force      = flags.Bool("force", false, "Skip rate limits (for resend-verification)")
		limit      = flags.Int("limit", 100, "Maximum results to show (for email-logs)")
	)

	if len(args) == 0 {
		printAuthUsage(stderr)
		return fmt.Errorf("missing auth subcommand")
	}

	subCmd := args[0]

	if err := flags.Parse(args[1:]); err != nil {
		printAuthUsage(stderr)
		return err
	}

	// Load config
	cfg, configFile, err := config.LoadWithPath(*configPath, getenv)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if !cfg.Auth.Enabled {
		return fmt.Errorf("authentication is not enabled in config")
	}

	dbPath := authDBPath(configFile)
	db, err := auth.OpenDB(dbPath)
	if err != nil {
		return fmt.Errorf("opening auth database: %w", err)
	}
	defer db.Close()

	// Execute subcommand
	switch subCmd {
	case "verify-email":
		if flags.NArg() == 0 {
			return fmt.Errorf("missing user ID")
		}
		return authVerifyEmailCmd(db, flags.Arg(0), stdout)
	case "resend-verification":
		if flags.NArg() == 0 {
			return fmt.Errorf("missing user ID")
		}
		return authResendVerificationCmd(db, cfg, flags.Arg(0), *force, stdout, stderr)
	case "reset-verification":
		if flags.NArg() == 0 {
			return fmt.Errorf("missing user ID")
		}
		return authResetVerificationCmd(db, flags.Arg(0), stdout)
	case "status":
		if flags.NArg() == 0 {
			return fmt.Errorf("missing user ID")
		}
		return authStatusCmd(db, flags.Arg(0), stdout)
	case "email-logs":
		userID := ""
		if flags.NArg() > 0 {
			userID = flags.Arg(0)
		}
		return authEmailLogsCmd(db, userID, *limit, stdout)
	default:
		printAuthUsage(stderr)
		return fmt.Errorf("unknown auth subcommand: %s", subCmd)
	}
}

func printAuthUsage(w io.Writer) {
	fmt.Fprintf(w, `basil auth - Manage email verification

Usage:
  basil auth <command> [options] [args]

Commands:
  verify-email <user_id>           Manually verify a user's email
  resend-verification <user_id>    Resend verification email
  reset-verification <user_id>     Clear verification state
  status <user_id>                 Show user verification status
  email-logs [user_id]             View email audit logs

Options:
  --config PATH     Path to config file
  --force           Skip rate limits (for resend-verification)
  --limit N         Maximum results (default: 100, for email-logs)

Examples:
  basil auth verify-email usr_abc123
  basil auth resend-verification usr_abc123
  basil auth resend-verification --force usr_abc123
  basil auth reset-verification usr_abc123
  basil auth status usr_abc123
  basil auth email-logs
  basil auth email-logs usr_abc123
  basil auth email-logs --limit 50

`)
}

// authVerifyEmailCmd manually verifies a user's email
func authVerifyEmailCmd(db *auth.DB, userID string, stdout io.Writer) error {
	user, err := db.GetUser(userID)
	if err != nil {
		return fmt.Errorf("getting user: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user not found: %s", userID)
	}

	if user.Email == "" {
		return fmt.Errorf("user has no email address")
	}

	if user.EmailVerifiedAt != nil {
		fmt.Fprintf(stdout, "✓ Email already verified: %s\n", user.Email)
		return nil
	}

	ctx := context.Background()
	if err := db.MarkEmailVerified(ctx, userID); err != nil {
		return fmt.Errorf("marking email verified: %w", err)
	}

	// Invalidate any pending verification tokens
	if err := db.InvalidateUserVerificationTokens(ctx, userID); err != nil {
		// Log but don't fail - verification succeeded
		fmt.Fprintf(stdout, "Warning: failed to invalidate pending tokens: %v\n", err)
	}

	fmt.Fprintf(stdout, "✓ Manually verified email: %s\n", user.Email)
	return nil
}

// authResendVerificationCmd resends a verification email
func authResendVerificationCmd(db *auth.DB, cfg *config.Config, userID string, force bool, stdout, stderr io.Writer) error {
	ctx := context.Background()

	user, err := db.GetUser(userID)
	if err != nil {
		return fmt.Errorf("getting user: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user not found: %s", userID)
	}

	if user.Email == "" {
		return fmt.Errorf("user has no email address")
	}

	if user.EmailVerifiedAt != nil {
		return fmt.Errorf("email already verified")
	}

	// Check email verification is enabled
	if !cfg.Auth.EmailVerification.Enabled {
		return fmt.Errorf("email verification is not enabled in config")
	}

	// Check rate limits unless --force
	if !force {
		result, err := db.CheckVerificationRateLimit(ctx, userID, user.Email,
			cfg.Auth.EmailVerification.ResendCooldown,
			cfg.Auth.EmailVerification.MaxSendsPerDay)
		if err != nil {
			return fmt.Errorf("checking rate limit: %w", err)
		}
		if !result.Allowed {
			if result.Reason == "cooldown" {
				return fmt.Errorf("rate limit: please wait until %s (use --force to bypass)", result.NextAvailable.Format("15:04:05"))
			}
			return fmt.Errorf("rate limit: %s (use --force to bypass)", result.Reason)
		}
	}

	// Build origin/baseURL for email links
	var origin string
	if cfg.Server.Dev {
		port := cfg.Server.Port
		if port == 0 || port == 443 {
			port = 8080
		}
		origin = fmt.Sprintf("http://localhost:%d", port)
	} else if cfg.Server.HTTPS.Auto || cfg.Server.HTTPS.Cert != "" {
		host := cfg.Server.Host
		if host == "" {
			host = "localhost"
		}
		origin = fmt.Sprintf("https://%s", host)
		if cfg.Server.Port != 443 {
			origin = fmt.Sprintf("%s:%d", origin, cfg.Server.Port)
		}
	} else {
		return fmt.Errorf("unable to determine base URL - configure HTTPS or use dev mode")
	}

	// Initialize email service
	emailService, err := auth.NewEmailService(&cfg.Auth.EmailVerification, db, origin)
	if err != nil {
		return fmt.Errorf("initializing email service: %w", err)
	}

	// Send verification email
	if err := emailService.SendVerificationEmail(ctx, user); err != nil {
		return fmt.Errorf("sending verification email: %w", err)
	}

	fmt.Fprintf(stdout, "✓ Sent verification email to: %s\n", user.Email)
	return nil
}

// authResetVerificationCmd clears verification state
func authResetVerificationCmd(db *auth.DB, userID string, stdout io.Writer) error {
	ctx := context.Background()

	user, err := db.GetUser(userID)
	if err != nil {
		return fmt.Errorf("getting user: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user not found: %s", userID)
	}

	// Unverify the email (direct SQL since there's no UnverifyEmail method)
	_, err = db.GetDB().Exec(`UPDATE users SET email_verified_at = NULL WHERE id = ?`, userID)
	if err != nil {
		return fmt.Errorf("resetting verification: %w", err)
	}

	// Invalidate pending tokens
	if err := db.InvalidateUserVerificationTokens(ctx, userID); err != nil {
		fmt.Fprintf(stdout, "Warning: failed to invalidate tokens: %v\n", err)
	}

	fmt.Fprintf(stdout, "✓ Reset verification state for: %s\n", user.Email)
	return nil
}

// authStatusCmd shows user verification status
func authStatusCmd(db *auth.DB, userID string, stdout io.Writer) error {
	user, err := db.GetUser(userID)
	if err != nil {
		return fmt.Errorf("getting user: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user not found: %s", userID)
	}

	fmt.Fprintf(stdout, "User: %s (%s)\n", user.Name, userID)
	fmt.Fprintf(stdout, "Email: %s\n", user.Email)

	if user.Email == "" {
		fmt.Fprintf(stdout, "Status: No email address\n")
		return nil
	}

	if user.EmailVerifiedAt != nil {
		fmt.Fprintf(stdout, "Status: ✓ Verified\n")
		fmt.Fprintf(stdout, "Verified at: %s\n", user.EmailVerifiedAt.Format("2006-01-02 15:04:05"))
	} else {
		fmt.Fprintf(stdout, "Status: ✗ Not verified\n")

		// Check for pending verification tokens
		rows, err := db.GetDB().Query(`
			SELECT id, created_at, expires_at, send_count, consumed_at
			FROM email_verifications
			WHERE user_id = ? AND consumed_at IS NULL
			ORDER BY created_at DESC
			LIMIT 1
		`, userID)
		if err != nil {
			return fmt.Errorf("checking tokens: %w", err)
		}
		defer rows.Close()

		if rows.Next() {
			var tokenID string
			var createdAt, expiresAt string
			var sendCount int
			var consumedAt *string

			if err := rows.Scan(&tokenID, &createdAt, &expiresAt, &sendCount, &consumedAt); err != nil {
				return fmt.Errorf("scanning token: %w", err)
			}

			fmt.Fprintf(stdout, "Pending token: %s\n", tokenID)
			fmt.Fprintf(stdout, "  Created: %s\n", createdAt)
			fmt.Fprintf(stdout, "  Expires: %s\n", expiresAt)
			fmt.Fprintf(stdout, "  Sends: %d\n", sendCount)
		}
	}

	return nil
}

// authEmailLogsCmd shows email audit logs
func authEmailLogsCmd(db *auth.DB, userID string, limit int, stdout io.Writer) error {
	ctx := context.Background()

	var logs []auth.EmailLog
	var err error

	if userID != "" {
		// Verify user exists
		user, err := db.GetUser(userID)
		if err != nil {
			return fmt.Errorf("getting user: %w", err)
		}
		if user == nil {
			return fmt.Errorf("user not found: %s", userID)
		}
		logs, err = db.GetEmailLogs(ctx, &userID, limit)
	} else {
		logs, err = db.GetEmailLogs(ctx, nil, limit)
	}

	if err != nil {
		return fmt.Errorf("getting email logs: %w", err)
	}

	if len(logs) == 0 {
		fmt.Fprintf(stdout, "No email logs found\n")
		return nil
	}

	// Print header
	fmt.Fprintf(stdout, "%-36s %-25s %-15s %-10s %-8s %s\n",
		"ID", "Recipient", "Type", "Provider", "Status", "Created")
	fmt.Fprintf(stdout, "%s\n", strings.Repeat("-", 120))

	// Print logs
	for _, log := range logs {
		userIDStr := ""
		if log.UserID != nil {
			userIDStr = *log.UserID
			if len(userIDStr) > 12 {
				userIDStr = userIDStr[:12]
			}
		}

		recipient := log.Recipient
		if len(recipient) > 25 {
			recipient = recipient[:22] + "..."
		}

		fmt.Fprintf(stdout, "%-36s %-25s %-15s %-10s %-8s %s\n",
			userIDStr,
			recipient,
			log.EmailType,
			log.Provider,
			log.Status,
			log.CreatedAt.Format("2006-01-02 15:04"))

		if log.Error != nil && *log.Error != "" {
			fmt.Fprintf(stdout, "  Error: %s\n", *log.Error)
		}
	}

	fmt.Fprintf(stdout, "\nShowing %d of up to %d logs\n", len(logs), limit)
	return nil
}
