package cmd

import (
	"encoding/json"
	"fmt"
	"summit/pkg/actions"
	"summit/pkg/config"
	"summit/pkg/diff"
	"summit/pkg/log"
	"summit/pkg/system"

	"github.com/spf13/cobra"
)

var (
	dryRun              bool
	applyPruneUnmanaged bool
)

// applyCmd represents the apply command
var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Applies the changes necessary to get the system to the desired state",
	Long: `The apply command reads the desired state from the system.yaml file
and applies the necessary changes to the Alpine Linux system to match that state.
It respects both intrinsic safety ignores and user-defined ignore patterns from the config.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load the configuration file
		logger := cmd.Context().Value("logger").(log.Logger)
		desiredSystemState, err := config.LoadConfig(cfgFile, logger)
		if err != nil {
			return err
		}

		// infer  system state
		currentSystemState, _, err := system.InferSystemState(cmdRunner, false)
		if err != nil {
			return err
		}

		plan, err := diff.CalculatePlan(desiredSystemState, currentSystemState, cmdRunner, applyPruneUnmanaged)
		if err != nil {
			return err
		}

		if dryRun {
			if jsonOutput {
				actionsForJSON := []actionForJSON{}
				for _, action := range plan {
					actionsForJSON = append(actionsForJSON, actionForJSON{
						Type:        fmt.Sprintf("%T", action),
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
				fmt.Fprintln(cmd.OutOrStdout(), "Dry run enabled. The following operations would be performed:")
				for _, action := range plan {
					fmt.Fprintf(cmd.OutOrStdout(), "=> %s\n", action.Description()) // Keep the high-level description
					details := action.ExecutionDetails()
					for _, detail := range details {
						fmt.Fprintf(cmd.OutOrStdout(), "   - %s\n", detail) // Print the detailed steps
					}
				}
			}
			return nil
		}

		// Execute the plan
		return executePlan(cmd, plan, cmdRunner, logger)
	},
}

func executePlan(cmd *cobra.Command, plan []actions.Action, runner system.CommandRunner, logger log.Logger) error {
	completedActions := []actions.Action{}

	for _, action := range plan {
		logger.Info(fmt.Sprintf("=> %s", action.Description()))
		if err := action.Apply(runner, logger); err != nil {
			logger.Error("Action failed, rolling back changes", "action", action.Description(), "error", err)
			rollbackPlan(cmd, completedActions, runner, logger)
			return err
		}
		completedActions = append(completedActions, action)
	}

	logger.Info("Apply complete.")
	return nil
}

func rollbackPlan(cmd *cobra.Command, plan []actions.Action, runner system.CommandRunner, logger log.Logger) {
	logger.Info("--- Starting Rollback ---")
	for i := len(plan) - 1; i >= 0; i-- {
		action := plan[i]
		logger.Info(fmt.Sprintf("<= Rolling back: %s", action.Description()))
		// We ignore the error here because the Rollback action itself is responsible for logging it.
		// We want to continue trying to roll back all other completed actions.
		_ = action.Rollback(runner, logger)
	}
	logger.Info("--- Rollback Complete ---")
}

func init() {
	rootCmd.AddCommand(applyCmd)
	applyCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what changes would be made without executing them")
	applyCmd.Flags().BoolVar(&applyPruneUnmanaged, "prune-unmanaged", false, "Delete unmanaged files not present in system.yaml")
	applyCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output the plan in JSON format (only valid with --dry-run)")
}
