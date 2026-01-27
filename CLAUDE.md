# CLAUDE.md

Project guidelines for Claude Code sessions working on `bak`.

## Session Workflow

### Branch Strategy
- **Create a new branch for big features** using the format: `claude/<brief-description>`
- Small fixes can be committed to an existing feature branch
- Never commit directly to `main`

```bash
# For new features
git checkout main
git pull origin main
git checkout -b claude/<description>
```

### Commits and Pushes
- Make small, focused commits with clear messages
- Commit after completing each logical unit of work
- Push regularly to preserve work and enable CI feedback
- Use conventional commit style: `feat:`, `fix:`, `refactor:`, `test:`, `docs:`, `chore:`

```bash
git add <specific-files>
git commit -m "feat: add validation for backup paths"
git push -u origin claude/<description>
```

### Pull Requests
- **Always create a PR after pushing changes and provide the PR link**
- Ensure CI passes before requesting merge
- PR description should summarize changes and link related issues

```bash
gh pr create --title "feat: description" --body "Summary of changes"
# Always share the PR URL with the user
```

## Go Best Practices

### Code Style
- Run `make fmt` before committing
- Run `make lint` to check for issues
- Follow standard Go conventions (effective go, code review comments)
- Keep functions focused and small
- Use meaningful variable and function names
- Handle errors explicitly - don't ignore them

### Package Structure
- `cmd/bak/` - CLI entry point only, minimal logic
- `internal/` - All business logic, not importable externally
- Keep packages focused on single responsibility

### Error Handling
- Wrap errors with context: `fmt.Errorf("failed to load config: %w", err)`
- Return errors to callers; let main decide how to handle
- Use `errors.Is()` and `errors.As()` for error checking

## Testing Requirements

### Test Coverage
- All new code must have tests
- All bug fixes must include a regression test
- Run `make test` before committing

### Test Style
- Use table-driven tests for multiple cases
- Test file goes next to source: `config.go` -> `config_test.go`
- Use `t.Helper()` in test helper functions
- Use `t.Parallel()` where safe

```go
func TestParseConfig(t *testing.T) {
    t.Parallel()
    tests := []struct {
        name    string
        input   string
        want    Config
        wantErr bool
    }{
        // test cases
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test logic
        })
    }
}
```

### What to Test
- Public functions in each package
- Error conditions and edge cases
- Configuration parsing and validation

## CI Requirements

CI must pass before merging. The pipeline runs:

1. **Lint** - `go fmt`, `go vet`, `staticcheck`
2. **Test** - `go test -race` with coverage
3. **Build** - Ensures binary compiles

### Before Pushing
```bash
make fmt      # Format code
make lint     # Check for issues
make test     # Run tests
make build    # Verify it builds
```

## Project Context

`bak` is a CLI wrapper for restic backups designed for homelab use. Key points:

- Uses Cobra for CLI framework
- Config stored at `/etc/backup/`
- Generates systemd units for scheduled backups
- Works with append-only backup servers

### Key Files
- `cmd/bak/main.go` - CLI commands and flags
- `internal/config/config.go` - Configuration loading/saving
- `internal/backup/backup.go` - Restic execution
- `internal/systemd/systemd.go` - Systemd unit generation
