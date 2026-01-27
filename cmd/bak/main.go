package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

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

	fmt.Printf("Setting up backup with tag: %s\n", setupTag)
	fmt.Printf("Paths: %v\n", paths)
	fmt.Printf("Schedule: %s\n", setupSchedule)
	fmt.Printf("Retention: hourly=%d, daily=%d, weekly=%d, monthly=%d, yearly=%d\n",
		setupKeepHourly, setupKeepDaily, setupKeepWeekly, setupKeepMonthly, setupKeepYearly)

	// TODO: Implement actual setup logic
	// - Check if config exists (require --force to overwrite)
	// - Create /etc/backup/backup.conf
	// - Create systemd service and timer
	// - Enable and start timer

	fmt.Println("\nSetup complete! (stub implementation)")
	return nil
}

func runEdit(cmd *cobra.Command, args []string) error {
	// TODO: Implement edit logic
	// - Load existing config
	// - Apply changes
	// - Show diff
	// - Require confirmation
	// - Update config and restart timer

	fmt.Println("Edit command not yet implemented")
	return nil
}

func runNow(cmd *cobra.Command, args []string) error {
	// TODO: Implement now logic
	// - Check if configured
	// - Run backup directly with live output

	fmt.Println("Running backup now... (stub implementation)")
	return nil
}

func runStatus(cmd *cobra.Command, args []string) error {
	// TODO: Implement status logic
	// - Show configuration
	// - Show systemd timer status
	// - Show recent snapshots
	// - Show next scheduled run

	fmt.Println("Status command not yet implemented")
	return nil
}

func runInternal(cmd *cobra.Command, args []string) error {
	// TODO: Implement internal backup logic for systemd
	// - Load env and config
	// - Execute restic backup
	// - Handle notifications

	fmt.Println("Running internal backup... (stub implementation)")
	return nil
}
