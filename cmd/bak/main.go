package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/magicmicky/bak/internal/backup"
	"github.com/magicmicky/bak/internal/config"
	"github.com/magicmicky/bak/internal/notify"
	"github.com/magicmicky/bak/internal/scheduler"
	"github.com/magicmicky/bak/internal/ui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var version = "dev"

// Global printer for colored output
var printer = ui.Default()

// Global scheduler for platform-specific task management
var sched = scheduler.New()

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:     "bak",
	Short:   "A simple wrapper for restic backups",
	Long:    `bak is a CLI tool that wraps restic to provide easy backup configuration and management for homelab environments.`,
	Version: version,
}

// Setup command flags
var (
	setupTag         string
	setupPaths       string
	setupSchedule    string
	setupKeepHourly  int
	setupKeepDaily   int
	setupKeepWeekly  int
	setupKeepMonthly int
	setupKeepYearly  int
	setupExcludes    []string
	setupNotify      string
	setupForce       bool
	setupDryRun      bool
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Configure automated backups for this host",
	Long: `Configure automated backups by creating a config file and scheduled task.

Example:
  sudo bak setup --tag webapp --paths /var/www,/etc/nginx
  sudo bak setup --tag gameserver --paths /opt/game/saves --schedule hourly --keep-hourly 24
  bak setup --tag test --paths /tmp --dry-run`,
	RunE: runSetup,
}

// Edit command flags
var (
	editPaths    string
	editSchedule string
	editExcludes []string
	editNotify   string
	editYes      bool
	editDryRun   bool
)

var editCmd = &cobra.Command{
	Use:   "edit",
	Short: "Modify existing backup configuration",
	Long: `Modify existing configuration without full reconfiguration.
Only specified flags will be changed.

Example:
  sudo bak edit --keep-daily 14 --keep-weekly 8
  sudo bak edit --paths /var/www,/etc/nginx,/opt/certs
  bak edit --schedule hourly --dry-run`,
	RunE: runEdit,
}

// Now command flags
var (
	nowDryRun  bool
	nowVerbose bool
)

var nowCmd = &cobra.Command{
	Use:   "now",
	Short: "Run backup immediately",
	Long: `Run backup immediately with live output. Requires prior configuration via 'bak setup'.

Use --dry-run to see what would be backed up without actually uploading any data.
Use --verbose to show each file being processed.`,
	RunE: runNow,
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show configuration and recent snapshots",
	Long:  `Display current configuration, scheduled task status, and recent snapshots.`,
	RunE:  runStatus,
}

var runInternalCmd = &cobra.Command{
	Use:    "run-internal",
	Short:  "Internal command called by scheduler (not for direct use)",
	Hidden: true,
	RunE:   runInternal,
}

// List command flags
var listLimit int

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List snapshots with restore hints",
	Long:  `List recent snapshots with their IDs and restore command hints.`,
	RunE:  runList,
}

// Logs command flags
var logsLines int

// Init command flags
var (
	initRepo     string
	initPassword string
	initForce    bool
	initDryRun   bool
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Show recent backup logs",
	Long:  `Display recent backup service logs.`,
	RunE:  runLogs,
}

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for the specified shell.

To load completions:

Bash:
  $ source <(bak completion bash)
  # Or add to ~/.bashrc:
  # eval "$(bak completion bash)"

Zsh:
  $ source <(bak completion zsh)
  # Or add to ~/.zshrc:
  # eval "$(bak completion zsh)"

Fish:
  $ bak completion fish | source
  # Or add to ~/.config/fish/completions/:
  # bak completion fish > ~/.config/fish/completions/bak.fish

PowerShell:
  PS> bak completion powershell | Out-String | Invoke-Expression
`,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	DisableFlagsInUseLine: true,
	RunE:                  runCompletion,
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize backup credentials",
	Long: `Configure repository credentials for backups. Run once per machine.

Example:
  sudo bak init --repo rest:https://user@backup.server:8000
  sudo bak init --repo rest:https://user@backup.server:8000 --password "secret"
  sudo RESTIC_REPOSITORY=rest:https://... RESTIC_PASSWORD=secret bak init
  sudo bak init --repo ... --password ... --dry-run`,
	RunE: runInit,
}

func init() {
	// Setup command flags
	setupCmd.Flags().StringVar(&setupTag, "tag", "", "Unique identifier for this host's backups (required)")
	setupCmd.Flags().StringVar(&setupPaths, "paths", "", "Comma-separated directories to backup (required)")
	setupCmd.Flags().StringVar(&setupSchedule, "schedule", "daily", "Schedule: daily, hourly, or cron expression")
	setupCmd.Flags().IntVar(&setupKeepHourly, "keep-hourly", 0, "Number of hourly snapshots to keep")
	setupCmd.Flags().IntVar(&setupKeepDaily, "keep-daily", 7, "Number of daily snapshots to keep")
	setupCmd.Flags().IntVar(&setupKeepWeekly, "keep-weekly", 4, "Number of weekly snapshots to keep")
	setupCmd.Flags().IntVar(&setupKeepMonthly, "keep-monthly", 6, "Number of monthly snapshots to keep")
	setupCmd.Flags().IntVar(&setupKeepYearly, "keep-yearly", 0, "Number of yearly snapshots to keep")
	setupCmd.Flags().StringArrayVar(&setupExcludes, "exclude", nil, "Exclude pattern (can be specified multiple times)")
	setupCmd.Flags().StringVar(&setupNotify, "notify", "", "Apprise notification URL")
	setupCmd.Flags().BoolVar(&setupForce, "force", false, "Overwrite existing configuration")
	setupCmd.Flags().BoolVar(&setupDryRun, "dry-run", false, "Show what would be written without making changes")
	setupCmd.MarkFlagRequired("tag")
	setupCmd.MarkFlagRequired("paths")

	// Setup command completions
	setupCmd.RegisterFlagCompletionFunc("schedule", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"daily", "hourly"}, cobra.ShellCompDirectiveNoFileComp
	})

	// Edit command flags
	editCmd.Flags().StringVar(&editPaths, "paths", "", "Update backup paths")
	editCmd.Flags().StringVar(&editSchedule, "schedule", "", "Update schedule")
	editCmd.Flags().StringArrayVar(&editExcludes, "exclude", nil, "Update exclude patterns")
	editCmd.Flags().StringVar(&editNotify, "notify", "", "Update notification URL")
	editCmd.Flags().BoolVar(&editYes, "yes", false, "Skip confirmation prompt")
	editCmd.Flags().BoolVar(&editDryRun, "dry-run", false, "Show what would be changed without making changes")

	// Integer flags for edit need special handling to detect if set
	editCmd.Flags().Int("keep-hourly", 0, "Update hourly retention")
	editCmd.Flags().Int("keep-daily", 0, "Update daily retention")
	editCmd.Flags().Int("keep-weekly", 0, "Update weekly retention")
	editCmd.Flags().Int("keep-monthly", 0, "Update monthly retention")
	editCmd.Flags().Int("keep-yearly", 0, "Update yearly retention")

	// Edit command completions
	editCmd.RegisterFlagCompletionFunc("schedule", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"daily", "hourly"}, cobra.ShellCompDirectiveNoFileComp
	})

	// Now command flags
	nowCmd.Flags().BoolVar(&nowDryRun, "dry-run", false, "Show what would be backed up without uploading data")
	nowCmd.Flags().BoolVarP(&nowVerbose, "verbose", "v", false, "Show each file being processed")

	// List command flags
	listCmd.Flags().IntVarP(&listLimit, "last", "n", 10, "Number of snapshots to show")

	// Logs command flags
	logsCmd.Flags().IntVarP(&logsLines, "lines", "n", 20, "Number of log lines to show")

	// Init command flags
	initCmd.Flags().StringVar(&initRepo, "repo", "", "Repository URL (falls back to RESTIC_REPOSITORY env var)")
	initCmd.Flags().StringVar(&initPassword, "password", "", "Repository password (falls back to RESTIC_PASSWORD env var)")
	initCmd.Flags().BoolVar(&initForce, "force", false, "Overwrite existing credentials")
	initCmd.Flags().BoolVar(&initDryRun, "dry-run", false, "Show what would be written without making changes")

	// Add commands to root
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(editCmd)
	rootCmd.AddCommand(nowCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(runInternalCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(completionCmd)
}

// validateTag checks if tag contains only alphanumeric characters, dashes, and underscores
func validateTag(tag string) error {
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, tag)
	if !matched {
		return fmt.Errorf("tag must contain only alphanumeric characters, dashes, and underscores")
	}
	return nil
}

// parsePaths splits comma-separated paths and validates them
func parsePaths(pathsStr string) ([]string, []string) {
	paths := strings.Split(pathsStr, ",")
	var validPaths []string
	var warnings []string

	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		validPaths = append(validPaths, p)
		if _, err := os.Stat(p); os.IsNotExist(err) {
			warnings = append(warnings, fmt.Sprintf("path does not exist: %s", p))
		}
	}
	return validPaths, warnings
}

func runSetup(cmd *cobra.Command, args []string) error {
	// Validate tag
	if err := validateTag(setupTag); err != nil {
		return err
	}

	// Parse and validate paths
	paths, warnings := parsePaths(setupPaths)
	if len(paths) == 0 {
		return fmt.Errorf("at least one valid path is required")
	}

	// Print warnings for non-existent paths
	for _, w := range warnings {
		printer.Warning("Warning: %s", w)
	}

	// Check if config already exists (only if not dry-run)
	if !setupDryRun && config.Exists(config.DefaultConfigPath) && !setupForce {
		return fmt.Errorf("configuration already exists at %s. Use --force to overwrite", config.DefaultConfigPath)
	}

	// Build configuration
	cfg := &config.Config{
		Tag:         setupTag,
		Paths:       paths,
		Excludes:    setupExcludes,
		KeepHourly:  setupKeepHourly,
		KeepDaily:   setupKeepDaily,
		KeepWeekly:  setupKeepWeekly,
		KeepMonthly: setupKeepMonthly,
		KeepYearly:  setupKeepYearly,
		NotifyURL:   setupNotify,
		Schedule:    setupSchedule,
	}

	// Dry run mode
	if setupDryRun {
		printer.Header("=== Dry Run Mode ===")
		printer.Info("")

		// Find the binary path for service generation
		binaryPath, err := os.Executable()
		if err != nil {
			binaryPath = sched.DefaultBinaryPath()
		}
		binaryPath, _ = filepath.Abs(binaryPath)

		printer.Header("Would write to %s:", config.DefaultConfigPath)
		printer.Info(strings.Repeat("-", 50))
		fmt.Print(cfg.Content())

		for _, f := range sched.DryRunInfo(cfg.Schedule, binaryPath) {
			printer.Info("")
			printer.Header("Would write to %s:", f.Path)
			printer.Info(strings.Repeat("-", 50))
			fmt.Print(f.Content)
		}

		printer.Info("")
		printer.Warning("No changes made.")
		return nil
	}

	// Create config directory
	configDir := filepath.Dir(config.DefaultConfigPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Save configuration
	if err := cfg.Save(config.DefaultConfigPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	printer.Info("Configuration saved to %s", config.DefaultConfigPath)

	// Install scheduled task
	if err := sched.Install(cfg.Schedule); err != nil {
		return fmt.Errorf("failed to install scheduled task: %w", err)
	}

	// Get next run time
	nextRun, err := sched.NextRun()
	if err != nil {
		nextRun = "(unknown)"
	}

	printer.Success("\nSetup complete!")
	printer.Info("  Tag:      %s", cfg.Tag)
	printer.Info("  Paths:    %s", strings.Join(cfg.Paths, ", "))
	printer.Info("  Schedule: %s", cfg.Schedule)
	printer.Info("  Retention: hourly=%d, daily=%d, weekly=%d, monthly=%d, yearly=%d",
		cfg.KeepHourly, cfg.KeepDaily, cfg.KeepWeekly, cfg.KeepMonthly, cfg.KeepYearly)
	printer.Info("  Next run: %s", strings.TrimSpace(nextRun))
	printer.Info("")
	printer.Warning("Note: Ensure %s contains RESTIC_REPOSITORY and RESTIC_PASSWORD", config.DefaultEnvPath)

	return nil
}

func runEdit(cmd *cobra.Command, args []string) error {
	// Check if configured
	if !config.Exists(config.DefaultConfigPath) {
		return fmt.Errorf("not configured. Run 'bak setup' first")
	}

	// Load existing config
	cfg, err := config.Load(config.DefaultConfigPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Track changes
	var changes []string
	oldSchedule := cfg.Schedule

	// Apply changes based on flags that were explicitly set
	if cmd.Flags().Changed("paths") {
		paths, warnings := parsePaths(editPaths)
		if len(paths) == 0 {
			return fmt.Errorf("at least one valid path is required")
		}
		for _, w := range warnings {
			printer.Warning("Warning: %s", w)
		}
		changes = append(changes, fmt.Sprintf("  paths: %s -> %s", strings.Join(cfg.Paths, ","), strings.Join(paths, ",")))
		cfg.Paths = paths
	}

	if cmd.Flags().Changed("schedule") {
		changes = append(changes, fmt.Sprintf("  schedule: %s -> %s", cfg.Schedule, editSchedule))
		cfg.Schedule = editSchedule
	}

	if cmd.Flags().Changed("exclude") {
		changes = append(changes, fmt.Sprintf("  excludes: %v -> %v", cfg.Excludes, editExcludes))
		cfg.Excludes = editExcludes
	}

	if cmd.Flags().Changed("notify") {
		changes = append(changes, fmt.Sprintf("  notify: %s -> %s", cfg.NotifyURL, editNotify))
		cfg.NotifyURL = editNotify
	}

	if cmd.Flags().Changed("keep-hourly") {
		val, _ := cmd.Flags().GetInt("keep-hourly")
		changes = append(changes, fmt.Sprintf("  keep-hourly: %d -> %d", cfg.KeepHourly, val))
		cfg.KeepHourly = val
	}

	if cmd.Flags().Changed("keep-daily") {
		val, _ := cmd.Flags().GetInt("keep-daily")
		changes = append(changes, fmt.Sprintf("  keep-daily: %d -> %d", cfg.KeepDaily, val))
		cfg.KeepDaily = val
	}

	if cmd.Flags().Changed("keep-weekly") {
		val, _ := cmd.Flags().GetInt("keep-weekly")
		changes = append(changes, fmt.Sprintf("  keep-weekly: %d -> %d", cfg.KeepWeekly, val))
		cfg.KeepWeekly = val
	}

	if cmd.Flags().Changed("keep-monthly") {
		val, _ := cmd.Flags().GetInt("keep-monthly")
		changes = append(changes, fmt.Sprintf("  keep-monthly: %d -> %d", cfg.KeepMonthly, val))
		cfg.KeepMonthly = val
	}

	if cmd.Flags().Changed("keep-yearly") {
		val, _ := cmd.Flags().GetInt("keep-yearly")
		changes = append(changes, fmt.Sprintf("  keep-yearly: %d -> %d", cfg.KeepYearly, val))
		cfg.KeepYearly = val
	}

	// Check if any changes were made
	if len(changes) == 0 {
		printer.Info("No changes specified. Use flags like --paths, --schedule, --keep-daily, etc.")
		return nil
	}

	// Dry run mode
	if editDryRun {
		printer.Header("=== Dry Run Mode ===")
		printer.Info("")
		printer.Header("Proposed changes:")
		for _, change := range changes {
			printer.Info(change)
		}

		printer.Info("")
		printer.Header("Would write to %s:", config.DefaultConfigPath)
		printer.Info(strings.Repeat("-", 50))
		fmt.Print(cfg.Content())

		if cfg.Schedule != oldSchedule {
			for _, f := range sched.DryRunInfo(cfg.Schedule, "") {
				printer.Info("")
				printer.Header("Would write to %s:", f.Path)
				printer.Info(strings.Repeat("-", 50))
				fmt.Print(f.Content)
			}
		}

		printer.Info("")
		printer.Warning("No changes made.")
		return nil
	}

	// Show diff
	printer.Header("Proposed changes:")
	for _, change := range changes {
		printer.Info(change)
	}

	// Confirm unless --yes is specified
	if !editYes {
		if !promptConfirm("\nApply these changes?") {
			printer.Info("Cancelled.")
			return nil
		}
	}

	// Save config
	if err := cfg.Save(config.DefaultConfigPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// If schedule changed, update the scheduled task
	if cfg.Schedule != oldSchedule {
		if err := sched.UpdateSchedule(cfg.Schedule); err != nil {
			return fmt.Errorf("failed to update schedule: %w", err)
		}
		printer.Info("Scheduled task updated.")
	}

	printer.Success("Configuration updated successfully.")
	return nil
}

// promptConfirm asks for user confirmation
func promptConfirm(prompt string) bool {
	fmt.Print(prompt + " [y/N]: ")
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	return strings.ToLower(strings.TrimSpace(response)) == "y"
}

func runNow(cmd *cobra.Command, args []string) error {
	// Check if restic is available
	if err := backup.RequireRestic(); err != nil {
		return fmt.Errorf("%w. Please install restic: https://restic.net/", err)
	}

	// Check if configured
	if !config.Exists(config.DefaultConfigPath) {
		return fmt.Errorf("not configured. Run 'bak setup' first")
	}

	// Load environment variables
	if err := config.LoadEnv(config.DefaultEnvPath); err != nil {
		return fmt.Errorf("failed to load environment: %w", err)
	}

	// Load configuration
	cfg, err := config.Load(config.DefaultConfigPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check repository is accessible
	printer.Info("Checking repository connection...")
	if err := backup.CheckRepository(); err != nil {
		return fmt.Errorf("cannot connect to repository. Check RESTIC_REPOSITORY and RESTIC_PASSWORD in %s", config.DefaultEnvPath)
	}

	// Run backup with verbose output
	if nowDryRun {
		printer.Header("=== Dry Run Mode ===")
		printer.Info("Starting dry-run backup for tag '%s'...", cfg.Tag)
	} else {
		printer.Header("Starting backup for tag '%s'...", cfg.Tag)
	}
	printer.Info("Paths: %s\n", strings.Join(cfg.Paths, ", "))

	runner := backup.NewRunner(cfg)
	runner.Verbose = nowVerbose
	runner.DryRun = nowDryRun
	if err := runner.Run(); err != nil {
		printer.Error("Backup failed!")
		return fmt.Errorf("backup failed: %w", err)
	}

	if nowDryRun {
		printer.Success("\nDry run completed. No data was uploaded.")
	} else {
		printer.Success("\nBackup completed successfully!")
	}
	return nil
}

func runStatus(cmd *cobra.Command, args []string) error {
	// Check if restic is available
	if err := backup.RequireRestic(); err != nil {
		return fmt.Errorf("%w. Please install restic: https://restic.net/", err)
	}

	// Check if configured
	if !config.Exists(config.DefaultConfigPath) {
		return fmt.Errorf("not configured. Run 'bak setup' first")
	}

	// Load configuration
	cfg, err := config.Load(config.DefaultConfigPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Display configuration
	printer.Header("=== Configuration ===")
	printer.Info("  Tag:      %s", cfg.Tag)
	printer.Info("  Paths:    %s", strings.Join(cfg.Paths, ", "))
	printer.Info("  Schedule: %s", cfg.Schedule)
	printer.Info("  Retention: hourly=%d, daily=%d, weekly=%d, monthly=%d, yearly=%d",
		cfg.KeepHourly, cfg.KeepDaily, cfg.KeepWeekly, cfg.KeepMonthly, cfg.KeepYearly)
	if len(cfg.Excludes) > 0 {
		printer.Info("  Excludes: %s", strings.Join(cfg.Excludes, ", "))
	}
	if cfg.NotifyURL != "" {
		printer.Info("  Notify:   %s", cfg.NotifyURL)
	}

	// Display scheduler status
	printer.Header("\n=== Scheduler Status ===")
	schedStatus, err := sched.Status()
	if err != nil {
		printer.Warning("  Scheduled task not installed or not running")
	} else {
		fmt.Println(schedStatus)
	}

	// Display next run time
	nextRun, err := sched.NextRun()
	if err == nil && strings.TrimSpace(nextRun) != "" {
		printer.Info("Next scheduled run: %s", strings.TrimSpace(nextRun))
	}

	// Display recent snapshots
	printer.Header("\n=== Recent Snapshots ===")
	if err := config.LoadEnv(config.DefaultEnvPath); err != nil {
		printer.Warning("  Cannot load environment: %v", err)
		printer.Info("  Ensure %s exists with RESTIC_REPOSITORY and RESTIC_PASSWORD", config.DefaultEnvPath)
		return nil
	}

	runner := backup.NewRunner(cfg)
	if err := runner.ListSnapshots(5); err != nil {
		printer.Warning("  Cannot list snapshots: %v", err)
		printer.Info("  Repository may be unreachable or not initialized")
	}

	return nil
}

func runInternal(cmd *cobra.Command, args []string) error {
	// Check if restic is available
	if err := backup.RequireRestic(); err != nil {
		return fmt.Errorf("%w. Please install restic: https://restic.net/", err)
	}

	// Load environment variables (RESTIC_REPOSITORY, RESTIC_PASSWORD, etc.)
	if err := config.LoadEnv(config.DefaultEnvPath); err != nil {
		return fmt.Errorf("failed to load environment: %w", err)
	}

	// Load configuration
	cfg, err := config.Load(config.DefaultConfigPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Run the backup
	runner := backup.NewRunner(cfg)
	backupErr := runner.Run()

	// Send notification if configured
	if cfg.NotifyURL != "" {
		notifier := notify.New(cfg.NotifyURL)
		if backupErr != nil {
			if notifyErr := notifier.SendFailure(cfg.Tag, backupErr); notifyErr != nil {
				fmt.Fprintf(os.Stderr, "Failed to send failure notification: %v\n", notifyErr)
			}
		} else {
			// Try to get the latest snapshot ID
			snapshotID := ""
			snapshots, listErr := runner.ListSnapshotsJSON(1)
			if listErr == nil && len(snapshots) > 0 {
				snapshotID = snapshots[0].ShortID
			}
			if notifyErr := notifier.SendSuccess(cfg.Tag, snapshotID); notifyErr != nil {
				fmt.Fprintf(os.Stderr, "Failed to send success notification: %v\n", notifyErr)
			}
		}
	}

	if backupErr != nil {
		return fmt.Errorf("backup failed: %w", backupErr)
	}

	return nil
}

func runList(cmd *cobra.Command, args []string) error {
	// Check if restic is available
	if err := backup.RequireRestic(); err != nil {
		return fmt.Errorf("%w. Please install restic: https://restic.net/", err)
	}

	// Check if configured
	if !config.Exists(config.DefaultConfigPath) {
		return fmt.Errorf("not configured. Run 'bak setup' first")
	}

	// Load environment variables
	if err := config.LoadEnv(config.DefaultEnvPath); err != nil {
		return fmt.Errorf("failed to load environment: %w", err)
	}

	// Load configuration
	cfg, err := config.Load(config.DefaultConfigPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get snapshots as JSON
	runner := backup.NewRunner(cfg)
	snapshots, err := runner.ListSnapshotsJSON(listLimit)
	if err != nil {
		return fmt.Errorf("failed to list snapshots: %w", err)
	}

	if len(snapshots) == 0 {
		printer.Warning("No snapshots found for tag '%s'", cfg.Tag)
		return nil
	}

	// Print header
	printer.Header("=== Snapshots for tag '%s' ===", cfg.Tag)
	printer.Info("%-10s %-20s %s", "ID", "Time", "Paths")

	// Print snapshots
	for _, s := range snapshots {
		// Parse and format time
		timeStr := s.Time
		if t, err := time.Parse(time.RFC3339Nano, s.Time); err == nil {
			timeStr = t.Format("2006-01-02 15:04")
		}

		pathsStr := strings.Join(s.Paths, ", ")
		if len(pathsStr) > 40 {
			pathsStr = pathsStr[:37] + "..."
		}

		printer.Info("%-10s %-20s %s", s.ShortID, timeStr, pathsStr)
	}

	// Print restore hint
	printer.Info("")
	printer.Success("To restore a snapshot:")
	printer.Info("  restic restore %s --target /path/to/restore", snapshots[0].ShortID)

	return nil
}

func runLogs(cmd *cobra.Command, args []string) error {
	return sched.ViewLogs(logsLines)
}

func runCompletion(cmd *cobra.Command, args []string) error {
	switch args[0] {
	case "bash":
		return cmd.Root().GenBashCompletion(os.Stdout)
	case "zsh":
		return cmd.Root().GenZshCompletion(os.Stdout)
	case "fish":
		return cmd.Root().GenFishCompletion(os.Stdout, true)
	case "powershell":
		return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
	}
	return nil
}

func runInit(cmd *cobra.Command, args []string) error {
	// Check if restic is available
	if err := backup.RequireRestic(); err != nil {
		return fmt.Errorf("%w. Please install restic: https://restic.net/", err)
	}

	// Get repository URL: flag → env → prompt
	repo := initRepo
	if repo == "" {
		repo = os.Getenv("RESTIC_REPOSITORY")
	}
	if repo == "" {
		repo = promptInput("Repository URL: ")
	}
	if repo == "" {
		return fmt.Errorf("repository URL is required")
	}

	// Get password: flag → env → prompt
	password := initPassword
	if password == "" {
		password = os.Getenv("RESTIC_PASSWORD")
	}
	if password == "" {
		password = promptPassword("Repository password: ")
	}
	if password == "" {
		return fmt.Errorf("repository password is required")
	}

	// Check if credentials already exist
	if !initDryRun && config.CredentialsExist() && !initForce {
		return fmt.Errorf("credentials already exist at %s. Use --force to overwrite", config.DefaultEnvPath)
	}

	// Validate credentials by running restic cat config
	printer.Info("Validating repository credentials...")
	if err := validateCredentials(repo, password); err != nil {
		return fmt.Errorf("cannot connect to repository: %w", err)
	}
	printer.Success("Repository connection verified.")

	// Dry run mode
	if initDryRun {
		printer.Header("\n=== Dry Run Mode ===")
		printer.Info("")
		printer.Header("Would write to %s:", config.DefaultEnvPath)
		printer.Info(strings.Repeat("-", 50))
		fmt.Printf(`# Restic credentials generated by bak init
RESTIC_REPOSITORY="%s"
RESTIC_PASSWORD_FILE="%s"
RESTIC_CACHE_DIR="%s"
`, repo, config.DefaultPasswordPath, config.DefaultCacheDir)

		printer.Info("")
		printer.Header("Would write to %s (mode 0600):", config.DefaultPasswordPath)
		printer.Info(strings.Repeat("-", 50))
		printer.Info("<password hidden>")

		printer.Info("")
		printer.Warning("No changes made.")
		return nil
	}

	// Write credentials
	if err := config.WriteCredentials(repo, password); err != nil {
		return fmt.Errorf("failed to write credentials: %w", err)
	}

	printer.Success("\nCredentials saved successfully!")
	printer.Info("  Env file:      %s", config.DefaultEnvPath)
	printer.Info("  Password file: %s", config.DefaultPasswordPath)
	printer.Info("")
	printer.Info("Next steps:")
	printer.Info("  sudo bak setup --tag <hostname> --paths /path/to/backup")

	return nil
}

// promptInput reads a line from stdin
func promptInput(prompt string) string {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return ""
	}
	return strings.TrimSpace(input)
}

// promptPassword reads a password from stdin with hidden input
func promptPassword(prompt string) string {
	fmt.Print(prompt)

	// Try to use terminal for hidden input
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		password, err := term.ReadPassword(fd)
		fmt.Println() // Add newline after hidden input
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(password))
	}

	// Fallback to regular input if terminal is not available
	return promptInput("")
}

// validateCredentials checks if the repository is accessible with the given credentials
func validateCredentials(repo, password string) error {
	cmd := exec.Command("restic", "cat", "config")
	cmd.Env = append(os.Environ(),
		"RESTIC_REPOSITORY="+repo,
		"RESTIC_PASSWORD="+password,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if len(output) > 0 {
			return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
		}
		return err
	}
	return nil
}
