package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/magicmicky/bak/internal/backup"
	"github.com/magicmicky/bak/internal/config"
	"github.com/magicmicky/bak/internal/systemd"
	"github.com/spf13/cobra"
)

var version = "dev"

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
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Configure automated backups for this host",
	Long: `Configure automated backups by creating a config file and systemd timer.

Example:
  sudo bak setup --tag webapp --paths /var/www,/etc/nginx
  sudo bak setup --tag gameserver --paths /opt/game/saves --schedule hourly --keep-hourly 24`,
	RunE: runSetup,
}

// Edit command flags
var (
	editPaths    string
	editSchedule string
	editExcludes []string
	editNotify   string
	editYes      bool
)

var editCmd = &cobra.Command{
	Use:   "edit",
	Short: "Modify existing backup configuration",
	Long: `Modify existing configuration without full reconfiguration.
Only specified flags will be changed.

Example:
  sudo bak edit --keep-daily 14 --keep-weekly 8
  sudo bak edit --paths /var/www,/etc/nginx,/opt/certs`,
	RunE: runEdit,
}

var nowCmd = &cobra.Command{
	Use:   "now",
	Short: "Run backup immediately",
	Long:  `Run backup immediately with live output. Requires prior configuration via 'bak setup'.`,
	RunE:  runNow,
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show configuration and recent snapshots",
	Long:  `Display current configuration, systemd timer status, and recent snapshots.`,
	RunE:  runStatus,
}

var runInternalCmd = &cobra.Command{
	Use:    "run-internal",
	Short:  "Internal command called by systemd (not for direct use)",
	Hidden: true,
	RunE:   runInternal,
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
	setupCmd.MarkFlagRequired("tag")
	setupCmd.MarkFlagRequired("paths")

	// Edit command flags
	editCmd.Flags().StringVar(&editPaths, "paths", "", "Update backup paths")
	editCmd.Flags().StringVar(&editSchedule, "schedule", "", "Update schedule")
	editCmd.Flags().StringArrayVar(&editExcludes, "exclude", nil, "Update exclude patterns")
	editCmd.Flags().StringVar(&editNotify, "notify", "", "Update notification URL")
	editCmd.Flags().BoolVar(&editYes, "yes", false, "Skip confirmation prompt")

	// Integer flags for edit need special handling to detect if set
	editCmd.Flags().Int("keep-hourly", 0, "Update hourly retention")
	editCmd.Flags().Int("keep-daily", 0, "Update daily retention")
	editCmd.Flags().Int("keep-weekly", 0, "Update weekly retention")
	editCmd.Flags().Int("keep-monthly", 0, "Update monthly retention")
	editCmd.Flags().Int("keep-yearly", 0, "Update yearly retention")

	// Add commands to root
	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(editCmd)
	rootCmd.AddCommand(nowCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(runInternalCmd)
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
			warnings = append(warnings, fmt.Sprintf("Warning: path does not exist: %s", p))
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
		fmt.Fprintln(os.Stderr, w)
	}

	// Check if config already exists
	if config.Exists(config.DefaultConfigPath) && !setupForce {
		return fmt.Errorf("configuration already exists at %s. Use --force to overwrite", config.DefaultConfigPath)
	}

	// Create config directory
	configDir := filepath.Dir(config.DefaultConfigPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
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

	// Save configuration
	if err := cfg.Save(config.DefaultConfigPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Configuration saved to %s\n", config.DefaultConfigPath)

	// Install systemd timer
	if err := systemd.Install(cfg.Schedule); err != nil {
		return fmt.Errorf("failed to install systemd timer: %w", err)
	}

	// Get next run time
	nextRun, err := systemd.NextRun()
	if err != nil {
		nextRun = "(unknown)"
	}

	fmt.Println("\nSetup complete!")
	fmt.Printf("  Tag:      %s\n", cfg.Tag)
	fmt.Printf("  Paths:    %s\n", strings.Join(cfg.Paths, ", "))
	fmt.Printf("  Schedule: %s\n", cfg.Schedule)
	fmt.Printf("  Retention: hourly=%d, daily=%d, weekly=%d, monthly=%d, yearly=%d\n",
		cfg.KeepHourly, cfg.KeepDaily, cfg.KeepWeekly, cfg.KeepMonthly, cfg.KeepYearly)
	fmt.Printf("  Next run: %s\n", strings.TrimSpace(nextRun))
	fmt.Printf("\nNote: Ensure %s contains RESTIC_REPOSITORY and RESTIC_PASSWORD\n", config.DefaultEnvPath)

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
			fmt.Fprintln(os.Stderr, w)
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
		fmt.Println("No changes specified. Use flags like --paths, --schedule, --keep-daily, etc.")
		return nil
	}

	// Show diff
	fmt.Println("Proposed changes:")
	for _, change := range changes {
		fmt.Println(change)
	}

	// Confirm unless --yes is specified
	if !editYes {
		if !promptConfirm("\nApply these changes?") {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Save config
	if err := cfg.Save(config.DefaultConfigPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// If schedule changed, update the timer
	if cfg.Schedule != oldSchedule {
		if err := systemd.WriteTimer(cfg.Schedule); err != nil {
			return fmt.Errorf("failed to update timer: %w", err)
		}
		if err := systemd.ReloadDaemon(); err != nil {
			return fmt.Errorf("failed to reload systemd: %w", err)
		}
		if err := systemd.RestartTimer(); err != nil {
			return fmt.Errorf("failed to restart timer: %w", err)
		}
		fmt.Println("Timer updated and restarted.")
	}

	fmt.Println("Configuration updated successfully.")
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
	fmt.Println("Checking repository connection...")
	if err := backup.CheckRepository(); err != nil {
		return fmt.Errorf("cannot connect to repository. Check RESTIC_REPOSITORY and RESTIC_PASSWORD in %s", config.DefaultEnvPath)
	}

	// Run backup with verbose output
	fmt.Printf("Starting backup for tag '%s'...\n", cfg.Tag)
	fmt.Printf("Paths: %s\n\n", strings.Join(cfg.Paths, ", "))

	runner := backup.NewRunner(cfg)
	runner.Verbose = true
	if err := runner.Run(); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	fmt.Println("\nBackup completed successfully!")
	return nil
}

func runStatus(cmd *cobra.Command, args []string) error {
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
	fmt.Println("=== Configuration ===")
	fmt.Printf("  Tag:      %s\n", cfg.Tag)
	fmt.Printf("  Paths:    %s\n", strings.Join(cfg.Paths, ", "))
	fmt.Printf("  Schedule: %s\n", cfg.Schedule)
	fmt.Printf("  Retention: hourly=%d, daily=%d, weekly=%d, monthly=%d, yearly=%d\n",
		cfg.KeepHourly, cfg.KeepDaily, cfg.KeepWeekly, cfg.KeepMonthly, cfg.KeepYearly)
	if len(cfg.Excludes) > 0 {
		fmt.Printf("  Excludes: %s\n", strings.Join(cfg.Excludes, ", "))
	}
	if cfg.NotifyURL != "" {
		fmt.Printf("  Notify:   %s\n", cfg.NotifyURL)
	}

	// Display timer status
	fmt.Println("\n=== Timer Status ===")
	timerStatus, err := systemd.TimerStatus()
	if err != nil {
		fmt.Println("  Timer not installed or not running")
	} else {
		fmt.Println(timerStatus)
	}

	// Display next run time
	nextRun, err := systemd.NextRun()
	if err == nil && strings.TrimSpace(nextRun) != "" {
		fmt.Printf("Next scheduled run: %s\n", strings.TrimSpace(nextRun))
	}

	// Display recent snapshots
	fmt.Println("\n=== Recent Snapshots ===")
	if err := config.LoadEnv(config.DefaultEnvPath); err != nil {
		fmt.Printf("  Cannot load environment: %v\n", err)
		fmt.Printf("  Ensure %s exists with RESTIC_REPOSITORY and RESTIC_PASSWORD\n", config.DefaultEnvPath)
		return nil
	}

	runner := backup.NewRunner(cfg)
	if err := runner.ListSnapshots(5); err != nil {
		fmt.Printf("  Cannot list snapshots: %v\n", err)
		fmt.Println("  Repository may be unreachable or not initialized")
	}

	return nil
}

func runInternal(cmd *cobra.Command, args []string) error {
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
	if err := runner.Run(); err != nil {
		// Log notification URL if configured (actual notification not implemented)
		if cfg.NotifyURL != "" {
			fmt.Fprintf(os.Stderr, "Backup failed. Notification URL configured: %s\n", cfg.NotifyURL)
		}
		return fmt.Errorf("backup failed: %w", err)
	}

	if cfg.NotifyURL != "" {
		fmt.Printf("Backup completed successfully. Notification URL configured: %s\n", cfg.NotifyURL)
	}

	return nil
}
