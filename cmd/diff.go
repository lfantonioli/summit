package cmd

import (
	"encoding/json"
	"fmt"
	"summit/pkg/config"
	"summit/pkg/diff"
	"summit/pkg/log"
	"summit/pkg/system"

	"github.com/spf13/cobra"
)

var diffPruneUnmanaged bool

// diffCmd represents the diff command
var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Shows the difference between the current state and the desired state",
	Long: `The diff command compares the current state of the Alpine Linux system
with the desired state defined in the system.yaml file and shows the differences.
It respects both intrinsic safety ignores and user-defined ignore patterns from the config.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := cmd.Context().Value("logger").(log.Logger)

		// Load the configuration file
		desiredSystemState, err := config.LoadConfig(cfgFile, logger)
		if err != nil {
			return err
		}

		// infer  system state
		currentSystemState, _, err := system.InferSystemState(cmdRunner, false)
		if err != nil {
			return err
		}

		// Generate the plan
		plan, err := diff.CalculatePlan(desiredSystemState, currentSystemState, cmdRunner, diffPruneUnmanaged)
		if err != nil {
			return err
		}

		if jsonOutput {
			actionsForJSON := []actionForJSON{}
			for _, action := range plan {
				actionsForJSON = append(actionsForJSON, actionForJSON{Type: fmt.Sprintf("%T", action),
					Description: action.Description(),
					Details:     action.ExecutionDetails(),
				})
			}
			jsonBytes, err := json.MarshalIndent(actionsForJSON, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal plan to JSON: %w", err)
			}
			fmt.Fprint(cmd.OutOrStdout(), string(jsonBytes))
		} else {
			// Print the plan
			fmt.Fprintln(cmd.OutOrStdout(), "The following operations will be performed:")
			for _, action := range plan {
				fmt.Fprintf(cmd.OutOrStdout(), "=> %s\n", action.Description()) // Keep the high-level description
				details := action.ExecutionDetails()
				for _, detail := range details {
					fmt.Fprintf(cmd.OutOrStdout(), "   - %s\n", detail) // Print the detailed steps
				}
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(diffCmd)
	diffCmd.Flags().BoolVar(&diffPruneUnmanaged, "prune-unmanaged", false, "Include deletion of unmanaged files in diff output")
	diffCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output the plan in JSON format")
}
