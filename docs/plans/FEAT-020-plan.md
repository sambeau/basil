# FEAT-020 Implementation Plan: Per-Developer Config Overrides

## Overview
Add a `developers` section to config that defines named developer instances. Each developer runs `basil --dev <name>` to use their config merged with the base.

## Current State
- `--dev` flag exists but only enables dev mode (HTTP, localhost)
- No concept of named developer configurations
- All developers share same config

## Implementation

### Step 1: Add DeveloperConfig struct
**File: `config/config.go`**

```go
// DeveloperConfig holds per-developer overrides
type DeveloperConfig struct {
    Port     int           `yaml:"port"`
    Database string        `yaml:"database"`
    Handlers string        `yaml:"handlers"`  // handlers root directory
    Static   string        `yaml:"static"`    // public_dir override
    Logging  LoggingConfig `yaml:"logging"`
}
```

Add to Config struct:
```go
type Config struct {
    // ... existing fields ...
    Developers map[string]DeveloperConfig `yaml:"developers"`
}
```

**Estimated: 15 minutes**

---

### Step 2: Add ApplyDeveloper function
**File: `config/load.go`**

```go
// ApplyDeveloper merges a named developer config onto the base config.
// Returns error if developer name not found.
func ApplyDeveloper(cfg *Config, name string) error {
    dev, ok := cfg.Developers[name]
    if !ok {
        // List available developers in error
        return fmt.Errorf("developer '%s' not found in config", name)
    }
    
    if dev.Port != 0 {
        cfg.Server.Port = dev.Port
    }
    if dev.Database != "" {
        cfg.Database.Path = dev.Database
    }
    if dev.Handlers != "" {
        // Need to update routes to use this base path
        // Or add HandlersRoot to config
    }
    if dev.Static != "" {
        cfg.PublicDir = dev.Static
    }
    // Logging: only override non-zero values
    if dev.Logging.Level != "" {
        cfg.Logging.Level = dev.Logging.Level
    }
    // etc.
    
    return nil
}
```

**Estimated: 30 minutes**

---

### Step 3: Modify --dev flag to accept name
**File: `cmd/basil/main.go`**

Current:
```go
devMode = flags.Bool("dev", false, "Development mode")
```

Change to:
```go
devName = flags.String("dev", "", "Development mode (optionally specify developer name)")
```

Then in logic:
```go
if *devName != "" {
    cfg.Server.Dev = true
    if *devName != "true" { // Handle bare --dev
        if err := config.ApplyDeveloper(cfg, *devName); err != nil {
            return err
        }
    }
}
```

Actually, this is tricky with `flag` package. Better approach:
```go
devMode = flags.Bool("dev", false, "Development mode (HTTP on localhost)")
devName = flags.String("as", "", "Run as named developer from config")
```

Usage: `basil --dev --as alice`

Or simpler - just use `--dev` for mode and add separate `--developer`:
```go
devMode    = flags.Bool("dev", false, "Development mode")
devProfile = flags.String("profile", "", "Developer profile name from config")
```

Usage: `basil --dev --profile alice`

**Recommendation**: `--profile` is clearer and avoids overloading `--dev`

**Estimated: 30 minutes**

---

### Step 4: Tests
**File: `config/load_test.go`**

- `TestApplyDeveloper` - merges correctly
- `TestApplyDeveloperNotFound` - error with message
- `TestApplyDeveloperPartial` - only overrides specified fields
- `TestDeveloperConfigParsing` - YAML parsing works

**File: `cmd/basil/main_test.go`**

- Test `--dev --profile alice` uses alice config

**Estimated: 45 minutes**

---

### Step 5: Documentation
- Update `basil.example.yaml` with developers section example
- Update CLI help text

**Estimated: 15 minutes**

---

## Total Estimated Effort
~2 hours

## Open Questions

1. **Flag name**: `--profile`, `--as`, `--developer`, or overload `--dev`?
   - Recommendation: `--profile` (clear, doesn't overload existing flag)

2. **Handlers path**: Add `HandlersRoot` to config, or keep per-route handlers?
   - Recommendation: Add `HandlersRoot` - simpler mental model

3. **`--dev` without `--profile` when developers exist**: Use first? Require profile?
   - Recommendation: Current behavior (dev mode with base config) - explicit is better

## Progress Log
| Step | Status | Notes |
|------|--------|-------|
| Step 1: DeveloperConfig struct | Not started | |
| Step 2: ApplyDeveloper function | Not started | |
| Step 3: --profile flag | Not started | |
| Step 4: Tests | Not started | |
| Step 5: Documentation | Not started | |
