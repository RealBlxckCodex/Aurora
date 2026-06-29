package cli

import (
	"fmt"

	"github.com/RealBlxckCodex/Aurora/internal/config"
	"github.com/RealBlxckCodex/Aurora/internal/models"
	"github.com/spf13/cobra"
)

func NewRemoveCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "rm <model>",
		Short: "Remove an installed model",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			modelID := args[0]
			_ = force

			cfg := config.Default()
			manager := models.NewManager(&cfg.Models)
			if err := manager.Initialize(); err != nil {
				return err
			}

			if err := manager.Remove(modelID); err != nil {
				return fmt.Errorf("remove failed: %w", err)
			}

			fmt.Printf("Model %s removed\n", modelID)
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Force removal")

	return cmd
}
