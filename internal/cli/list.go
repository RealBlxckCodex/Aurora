package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/RealBlxckCodex/Aurora/internal/config"
	"github.com/RealBlxckCodex/Aurora/internal/models"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func NewListCmd() *cobra.Command {
	var (
		modelType    string
		installed    bool
		format       string
		manifestPath string
		registryURL  string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available and installed models",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Default()
			manager := models.NewManager(&cfg.Models)
			if err := manager.Initialize(); err != nil {
				return err
			}

			allModels := manager.List()

			if registryURL == "" {
				registryURL = cfg.Models.RegistryURL
			}

			var manifest *models.Manifest

			if manifestPath != "" {
				var err error
				manifest, err = models.LoadManifest(manifestPath)
				if err == nil {
					goto display
				}
			}

			{
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				var err error
				manifest, err = models.FetchManifest(ctx, registryURL)
				if err == nil {
					goto display
				}
			}

			if manifestPath == "" {
				if _, err := os.Stat(models.DefaultManifestPath()); err == nil {
					manifest, _ = models.LoadManifest(models.DefaultManifestPath())
				}
			}

		display:
			type displayModel struct {
				ID        string
				Type      string
				Format    string
				Installed bool
				Size      int64
				Version   string
			}

			modelMap := make(map[string]*displayModel)
			for _, m := range allModels {
				modelMap[m.ID] = &displayModel{
					ID:        m.ID,
					Type:      string(m.Type),
					Format:    string(m.Format),
					Installed: m.Installed,
				}
			}

			if manifest != nil {
				for id, entry := range manifest.Models {
					if _, ok := modelMap[id]; !ok {
						totalSize := int64(0)
						for _, f := range entry.Files {
							totalSize += f.Size
						}
						modelMap[id] = &displayModel{
							ID:     id,
							Type:   entry.Type,
							Format: entry.Format,
							Version: entry.Version,
							Size:   totalSize,
						}
					}
				}
			}

			var filtered []*displayModel
			for _, dm := range modelMap {
				if modelType != "" && dm.Type != modelType {
					continue
				}
				if installed && !dm.Installed {
					continue
				}
				filtered = append(filtered, dm)
			}

			if format == "json" {
				enc := yaml.NewEncoder(os.Stdout)
				defer enc.Close()
				return enc.Encode(map[string]interface{}{"models": filtered})
			}

			if len(filtered) == 0 {
				fmt.Println("No models found")
				return nil
			}

			fmt.Printf("%-30s %-6s %-8s %-10s %s\n", "ID", "TYPE", "FORMAT", "SIZE", "STATUS")
			for _, dm := range filtered {
				status := "available"
				if dm.Installed {
					status = "installed"
				}
				sizeStr := ""
				if dm.Size > 0 {
					sizeStr = fmt.Sprintf("%.1fGB", float64(dm.Size)/1024/1024/1024)
				}
				fmt.Printf("%-30s %-6s %-8s %-10s %s\n", dm.ID, dm.Type, dm.Format, sizeStr, status)
			}

			if manifest != nil {
				fmt.Printf("\nUse 'aurora pull <model>' to download\n")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&modelType, "type", "", "Filter by type (tts|stt)")
	cmd.Flags().BoolVar(&installed, "installed", false, "Show only installed")
	cmd.Flags().StringVar(&format, "format", "table", "Output format (table|json)")
	cmd.Flags().StringVar(&manifestPath, "manifest", "", "Path to local manifest.json")
	cmd.Flags().StringVar(&registryURL, "registry", "", "Registry URL (default: from config)")

	return cmd
}
