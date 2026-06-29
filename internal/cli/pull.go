package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/RealBlxckCodex/Aurora/internal/config"
	"github.com/RealBlxckCodex/Aurora/internal/models"
	"github.com/spf13/cobra"
)

func NewPullCmd() *cobra.Command {
	var (
		quantization string
		manifestPath string
		modelDir     string
		registryURL  string
	)

	cmd := &cobra.Command{
		Use:   "pull <model>[:<version>]",
		Short: "Download a model from the registry",
		Long: `Download a model from the Aurora registry or HuggingFace.

Registry:   aurora pull <model>[:<version>]
HuggingFace: aurora pull hf.co/<namespace>/<repo>:<file>

The registry URL defaults to the config value (http://localhost:8000).
Override with --registry or set models.registry_url in config.yaml.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			arg := args[0]
			_ = quantization

			if isHuggingFaceRef(arg) {
				return pullFromHuggingFace(arg, modelDir)
			}

			modelID := arg
			requestedVersion := ""

			if idx := strings.LastIndex(arg, ":"); idx > 0 && !strings.HasPrefix(arg, ":") {
				modelID = arg[:idx]
				requestedVersion = arg[idx+1:]
			}

			cfg := config.Default()

			cfgDir := cfg.Models.Dir
			if cfgDir == "" {
				cfgDir = "/var/aurora/models"
			}
			if modelDir == "" {
				modelDir = cfgDir
			}

			registryURL = firstNonEmpty(registryURL, cfg.Models.RegistryURL, "http://localhost:8000")

			var manifest *models.Manifest

			if manifestPath != "" {
				var err error
				manifest, err = models.LoadManifest(manifestPath)
				if err != nil {
					return fmt.Errorf("load local manifest: %w", err)
				}
			} else {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				var err error
				manifest, err = models.FetchManifest(ctx, registryURL)
				if err != nil {
					return fmt.Errorf("cannot reach registry at %s: %w", registryURL, err)
				}
			}

			entry, ok := manifest.GetModel(modelID)
			if !ok {
				return fmt.Errorf("model %q not found in registry", modelID)
			}

			if requestedVersion != "" && entry.Version != requestedVersion {
				return fmt.Errorf("version %q not available for %s (latest: %s)", requestedVersion, modelID, entry.Version)
			}

			destDir := filepath.Join(modelDir, modelID)
			if err := os.MkdirAll(destDir, 0755); err != nil {
				return fmt.Errorf("create dir: %w", err)
			}

			dl := models.NewDownloader()

			for filename, fileEntry := range entry.Files {
				destPath := filepath.Join(destDir, filename)

				if _, err := os.Stat(destPath); err == nil {
					fmt.Printf("  %s already exists, skipping\n", filename)
					continue
				}

				url := fileEntry.URL
				if url == "" {
					url = manifest.BaseURL + "/" + modelID + "/" + filename
				}

				fmt.Printf("  Downloading %s... (%d MB)\n", filename, fileEntry.Size/1024/1024)

				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
				defer cancel()

				if err := dl.Download(ctx, url, destPath, fileEntry.SHA256); err != nil {
					return fmt.Errorf("download %s failed: %w", filename, err)
				}

				fmt.Printf("  %s saved to %s\n", filename, destPath)
			}

			fmt.Printf("Model %q installed successfully in %s\n", modelID, destDir)
			fmt.Printf("  Type: %s\n", entry.Type)
			fmt.Printf("  Format: %s\n", entry.Format)
			if len(entry.Voices) > 0 {
				fmt.Printf("  Voices: %v\n", entry.Voices)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&quantization, "quantization", "", "Quantization (q4_0, q8_0, fp16)")
	cmd.Flags().StringVar(&manifestPath, "manifest", "", "Path to local manifest.json")
	cmd.Flags().StringVar(&registryURL, "registry", "", "Registry URL (default: from config)")
	cmd.Flags().StringVar(&modelDir, "model-dir", "", "Install directory (default: from config)")

	return cmd
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func isHuggingFaceRef(arg string) bool {
	return strings.HasPrefix(arg, "hf.co/") || strings.HasPrefix(arg, "huggingface.co/")
}

func pullFromHuggingFace(arg, modelDir string) error {
	ref := arg
	stripPrefix := "hf.co/"
	if strings.HasPrefix(ref, "huggingface.co/") {
		stripPrefix = "huggingface.co/"
	}
	ref = strings.TrimPrefix(ref, stripPrefix)

	parts := strings.SplitN(ref, ":", 2)
	repoPath := parts[0]
	filename := ""
	if len(parts) == 2 {
		filename = parts[1]
	}

	repoParts := strings.SplitN(repoPath, "/", 2)
	if len(repoParts) != 2 {
		return fmt.Errorf("invalid HuggingFace reference %q, expected namespace/model", arg)
	}
	namespace, repo := repoParts[0], repoParts[1]

	if modelDir == "" {
		modelDir = "/var/aurora/models"
	}

	modelID := "hf/" + namespace + "/" + repo
	if filename != "" {
		modelID += "/" + filename
	}

	downloadURL := "https://huggingface.co/" + namespace + "/" + repo + "/resolve/main/" + filename

	destDir := filepath.Join(modelDir, modelID)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	destPath := filepath.Join(destDir, filename)
	if _, err := os.Stat(destPath); err == nil {
		fmt.Printf("  %s already exists, skipping\n", filename)
		fmt.Printf("Model already installed at %s\n", destDir)
		return nil
	}

	fmt.Printf("  Downloading %s from HuggingFace...\n", filename)
	fmt.Printf("    URL: %s\n", downloadURL)

	dl := models.NewDownloader()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	if err := dl.Download(ctx, downloadURL, destPath, ""); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	fmt.Printf("  %s saved to %s\n", filename, destPath)
	fmt.Printf("Model %q installed successfully in %s\n", modelID, destDir)

	return nil
}
