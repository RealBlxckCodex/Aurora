package cli

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
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
		quantization  string
		manifestPath  string
		modelDir      string
		registryURL   string
		releaseTag    string
		installAll    bool
	)

	cmd := &cobra.Command{
		Use:   "pull <model>[:<version>]",
		Short: "Download a model from the registry, GitHub release, or HuggingFace",
		Long: `Download a model from the Aurora registry, GitHub Release, or HuggingFace.

Registry:     aurora pull <model>[:<version>]
GitHub Release: aurora pull <model> --release <tag>
HuggingFace:  aurora pull hf.co/<namespace>/<repo>:<file>

The registry URL defaults to the config value (http://localhost:8000).`,
		Args: cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if releaseTag != "" && len(args) == 0 && !installAll {
				return fmt.Errorf("specify a model to pull, or use --all")
			}

			cfg := config.Default()
			modelDir = firstNonEmpty(modelDir, cfg.Models.Dir, "/var/aurora/models")

			if err := checkWritable(modelDir); err != nil {
				return fmt.Errorf("%w\n  → run with sudo or set a custom model dir in config (e.g. ~/.aurora/models)", err)
			}

			if releaseTag != "" {
				return pullFromRelease(releaseTag, args, modelDir)
			}

			if installAll {
				return pullAll(modelDir)
			}

			if len(args) == 0 {
				return fmt.Errorf("specify a model to pull, or use --all")
			}

			arg := args[0]

			if isHuggingFaceRef(arg) {
				return pullFromHuggingFace(arg, modelDir)
			}

			modelID := arg
			requestedVersion := ""

			if idx := strings.LastIndex(arg, ":"); idx > 0 && !strings.HasPrefix(arg, ":") {
				modelID = arg[:idx]
				requestedVersion = arg[idx+1:]
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

			return pullFromManifest(manifest, modelID, requestedVersion, modelDir)
		},
	}

	cmd.Flags().StringVar(&quantization, "quantization", "", "Quantization (q4_0, q8_0, fp16)")
	cmd.Flags().StringVar(&manifestPath, "manifest", "", "Path to local manifest.json")
	cmd.Flags().StringVar(&registryURL, "registry", "", "Registry URL (default: from config)")
	cmd.Flags().StringVar(&modelDir, "model-dir", "", "Install directory (default: from config)")
	cmd.Flags().StringVar(&releaseTag, "release", "", "Pull from GitHub Release tag (e.g. v0.1.0)")
	cmd.Flags().BoolVar(&installAll, "all", false, "Install all available models from manifest")

	return cmd
}

func pullFromManifest(manifest *models.Manifest, modelID, requestedVersion, modelDir string) error {
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
}

func pullFromRelease(tag string, args []string, modelDir string) error {
	baseURL := fmt.Sprintf("https://github.com/RealBlxckCodex/Aurora/releases/download/%s", tag)

	// Fetch manifest from release
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	manifestURL := baseURL + "/manifest.json"
	manifest, err := models.FetchManifestRaw(ctx, manifestURL)
	if err != nil {
		// Fallback: use the manifest from the repo
		manifest, err = models.FetchManifestRaw(ctx, "https://raw.githubusercontent.com/RealBlxckCodex/Aurora/"+tag+"/models/manifest.json")
		if err != nil {
			// Last resort: use local manifest
			manifest, err = models.LoadManifest("models/manifest.json")
			if err != nil {
				return fmt.Errorf("cannot load manifest from release, GitHub, or local: %w", err)
			}
		}
	}

	if len(args) == 0 {
		// Install all models
		for modelID := range manifest.Models {
			if err := installReleaseModel(manifest, modelID, baseURL, modelDir); err != nil {
				fmt.Printf("  Error installing %s: %v\n", modelID, err)
			}
		}
		return nil
	}

	for _, modelID := range args {
		if err := installReleaseModel(manifest, modelID, baseURL, modelDir); err != nil {
			return fmt.Errorf("install %s: %w", modelID, err)
		}
	}

	return nil
}

func pullAll(modelDir string) error {
	manifest, err := models.LoadManifest("models/manifest.json")
	if err != nil {
		return fmt.Errorf("load local manifest: %w", err)
	}
	baseURL := manifest.BaseURL

	for modelID := range manifest.Models {
		if err := installReleaseModel(manifest, modelID, baseURL, modelDir); err != nil {
			fmt.Printf("  ✗ %s: %v\n", modelID, err)
		} else {
			fmt.Printf("  ✓ %s\n", modelID)
		}
	}
	return nil
}

func installReleaseModel(manifest *models.Manifest, modelID, baseURL, modelDir string) error {
	entry, ok := manifest.GetModel(modelID)
	if !ok {
		return fmt.Errorf("model %q not found in manifest", modelID)
	}

	destDir := filepath.Join(modelDir, modelID)
	if _, err := os.Stat(destDir); err == nil {
		// Check if any expected files exist
		allExist := true
		for filename := range entry.Files {
			if _, err := os.Stat(filepath.Join(destDir, filename)); os.IsNotExist(err) {
				allExist = false
				break
			}
		}
		if allExist {
			fmt.Printf("Model %q already installed, skipping\n", modelID)
			return nil
		}
	}

	fmt.Printf("Installing %s...\n", modelID)

	// Check if model has HF URLs (download directly)
	allHaveCustomURLs := true
	for _, fileEntry := range entry.Files {
		if fileEntry.URL == "" {
			allHaveCustomURLs = false
			break
		}
	}

	if allHaveCustomURLs {
		// Install from direct URLs (HF)
		return pullFromManifest(manifest, modelID, "", modelDir)
	}

	// Download tarball from GitHub Release
	bundleName := entry.Bundle
	if bundleName == "" {
		bundleName = "models-" + modelID + ".tar.gz"
	}
	bundleURL := baseURL + "/" + bundleName

	tmpDir, err := os.MkdirTemp("", "aurora-bundle-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	tmpArchive := filepath.Join(tmpDir, bundleName)

	fmt.Printf("  Downloading %s...\n", bundleURL)

	dl := models.NewDownloader()
	dlCtx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	bundleSHA := entry.BundleSha256
	if err := dl.Download(dlCtx, bundleURL, tmpArchive, bundleSHA); err != nil {
		return fmt.Errorf("download bundle: %w", err)
	}

	fmt.Printf("  Extracting to %s...\n", destDir)

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}
	if err := extractTarGz(tmpArchive, destDir); err != nil {
		return fmt.Errorf("extract bundle: %w", err)
	}

	// Verify extracted files
	for filename, fileEntry := range entry.Files {
		destPath := filepath.Join(destDir, filename)
		if _, err := os.Stat(destPath); os.IsNotExist(err) {
			return fmt.Errorf("expected file %s not found in bundle", filename)
		}
		if fileEntry.SHA256 != "" {
			got, err := models.FileSHA256(destPath)
			if err != nil {
				return fmt.Errorf("checksum %s: %w", filename, err)
			}
			if got != fileEntry.SHA256 {
				return fmt.Errorf("sha256 mismatch for %s: expected %s, got %s", filename, fileEntry.SHA256, got)
			}
		}
	}

	fmt.Printf("  Model %q installed in %s\n", modelID, destDir)
	return nil
}

func extractTarGz(src, destDir string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(destDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			of, err := os.Create(target)
			if err != nil {
				return err
			}
			if _, err := io.Copy(of, tr); err != nil {
				of.Close()
				return err
			}
			of.Close()
		case tar.TypeSymlink:
			// Skip symlinks in bundles
			continue
		}
	}

	return nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func checkWritable(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("cannot create model directory %q: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, ".write-test-*")
	if err != nil {
		return fmt.Errorf("cannot write to %q: %w", dir, err)
	}
	os.Remove(tmp.Name())
	return nil
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

	modelID := filename
	if modelID == "" {
		modelID = namespace + "-" + repo
	}
	modelID = strings.TrimSuffix(modelID, ".gguf")
	modelID = strings.TrimSuffix(modelID, ".bin")
	modelID = strings.TrimSuffix(modelID, ".onnx")

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
