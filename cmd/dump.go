package cmd

import (
	"encoding/json"
	"fmt"
	"summit/pkg/config"
	"summit/pkg/diff"
	"summit/pkg/log"
	"summit/pkg/model"
	"summit/pkg/system"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// dumpCmd represents the dump command
var (
	dumpShowIgnored    bool
	dumpPreviewIgnores string
	dumpRaw            bool
	dumpAllServices    bool
)

func previewIgnoresFunc(cmd *cobra.Command, configFile string, logger log.Logger) error {
	// Load the config
	cfg, err := config.LoadConfig(configFile, logger)
	if err != nil {
		return fmt.Errorf("error loading config %s: %w", configFile, err)
	}

	// Get all configs (including ignored)
	allState, _, err := system.InferSystemState(cmdRunner, true) // skip intrinsic ignores
	if err != nil {
		return err
	}

	// Find which would be ignored by config patterns
	var wouldIgnore []string
	for _, conf := range allState.Configs {
		for _, pattern := range cfg.IgnoredConfigs {
			if diff.MatchesGlob(pattern, conf.Path) {
				wouldIgnore = append(wouldIgnore, conf.Path)
				break
			}
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Files that would be ignored by %s:\n", configFile)
	for _, path := range wouldIgnore {
		fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", path)
	}
	if len(wouldIgnore) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "  (none)")
	}

	return nil
}

var dumpCmd = &cobra.Command{
	Use:   "dump",
	Short: "Dumps the current system state to the console",
	Long: `The dump command reads the current system state and prints it to the console in YAML format.

It shows the actual state of the system, excluding intrinsically ignored files for security/safety.
By default, only enabled services are shown. Use --all-services to see all available services.
Use --show-ignored to see what files are ignored and why.
Use --preview-ignores <config> to see what would be ignored by a config file.
Use --raw to show all files including security-sensitive ones (use with caution).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := cmd.Context().Value("logger").(log.Logger)

		// Handle --preview-ignores
		if dumpPreviewIgnores != "" {
			return previewIgnoresFunc(cmd, dumpPreviewIgnores, logger)
		}

		// infer system state
		currentSystemState, ignored, err := system.InferSystemState(cmdRunner, dumpRaw)
		if err != nil {
			return err
		}

		// Filter out disabled services (not in any runlevel) unless --all-services is specified
		if !dumpAllServices {
			filteredServices := []model.ServiceState{}
			for _, svc := range currentSystemState.Services {
				// Keep services that are enabled or have a runlevel
				if svc.Enabled || svc.Runlevel != "" {
					filteredServices = append(filteredServices, svc)
				}
			}
			currentSystemState.Services = filteredServices
		}

		if jsonOutput {
			jsonData, err := json.MarshalIndent(currentSystemState, "", "  ")
			if err != nil {
				return fmt.Errorf("error marshaling to JSON: %w", err)
			}
			fmt.Fprint(cmd.OutOrStdout(), string(jsonData))
		} else {
			// Marshal the system state to YAML
			yamlData, err := yaml.Marshal(currentSystemState)
			if err != nil {
				return fmt.Errorf("error marshaling to YAML: %w", err)
			}
			// Print the YAML to the console
			fmt.Fprint(cmd.OutOrStdout(), string(yamlData))
		}

		// Show ignored files if requested
		if dumpShowIgnored && len(ignored) > 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "\n# Ignored files:")
			for _, ig := range ignored {
				fmt.Fprintf(cmd.OutOrStdout(), "#   %s (%s)\n", ig.Path, ig.Reason)
			}
		}

		// Warning for raw dump
		if dumpRaw {
			fmt.Fprintln(cmd.OutOrStdout(), "\n# Warning: --raw mode shows all files including security-sensitive ones")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(dumpCmd)
	dumpCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output the state in JSON format")
	dumpCmd.Flags().BoolVar(&dumpShowIgnored, "show-ignored", false, "Show files that are ignored with reasons")
	dumpCmd.Flags().StringVar(&dumpPreviewIgnores, "preview-ignores", "", "Preview which files would be ignored by the specified config file")
	dumpCmd.Flags().BoolVar(&dumpRaw, "raw", false, "Show all files including security-sensitive ones (use with caution)")
	dumpCmd.Flags().BoolVar(&dumpAllServices, "all-services", false, "Show all services including those not enabled in any runlevel")
}
