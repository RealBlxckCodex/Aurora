package main

import (
	"fmt"
	"os"

	"github.com/RealBlxckCodex/Aurora/internal/cli"
	"github.com/spf13/cobra"
)

var Version = "1.0.0"

func main() {
	root := &cobra.Command{
		Use:   "aurora",
		Short: "Aurora – Self-hosted Audio Inference Engine",
		Long: `Aurora is a fully self-hosted, CPU-first audio inference engine 
for TTS and STT with an OpenAI-compatible API.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			version, _ := cmd.Flags().GetBool("version")
			if version {
				fmt.Println("Aurora version", Version)
				return nil
			}
			return cmd.Help()
		},
	}

	root.Flags().Bool("version", false, "Print version and exit")

	root.AddCommand(cli.NewServeCmd())
	root.AddCommand(cli.NewPullCmd())
	root.AddCommand(cli.NewListCmd())
	root.AddCommand(cli.NewRemoveCmd())
	root.AddCommand(cli.NewG2PCmd())

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
