package cli

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/RealBlxckCodex/Aurora/internal/api"
	"github.com/RealBlxckCodex/Aurora/internal/config"
	"github.com/RealBlxckCodex/Aurora/internal/hardware"
	"github.com/RealBlxckCodex/Aurora/internal/inference"
	"github.com/RealBlxckCodex/Aurora/internal/models"
	"github.com/RealBlxckCodex/Aurora/pkg/domain"
	"github.com/spf13/cobra"
)

func NewServeCmd() *cobra.Command {
	var (
		host       string
		port       int
		cpuOnly    bool
		cpuThreads int
		configPath string
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the Aurora API server",
		Long:  "Start the Aurora audio inference API server with OpenAI-compatible endpoints",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Default()
			if configPath != "" {
				var err error
				cfg, err = config.Load(configPath)
				if err != nil {
					return fmt.Errorf("load config: %w", err)
				}
			}

			if host != "" {
				cfg.Server.Host = host
			}
			if port != 0 {
				cfg.Server.Port = port
			}
			if cpuThreads > 0 {
				cfg.Hardware.CPU.Threads = cpuThreads
			}
			if cpuOnly {
				cfg.Hardware.GPU.Enabled = "false"
			}

			cpuBackend, err := hardware.NewCPUBackend()
			if err != nil {
				return fmt.Errorf("init cpu: %w", err)
			}

			modelDir := cfg.Models.Dir

			modelManager := models.NewManager(&cfg.Models)
			if err := modelManager.Initialize(); err != nil {
				return fmt.Errorf("init model manager: %w", err)
			}
			_ = modelManager

			backendConfig := inference.BackendConfig{
				ModelDir: modelDir,
				Threads:  cfg.Hardware.CPU.Threads,
			}

			kokoro := inference.NewKokoroBackend()
			kokoro.Initialize(inference.BackendConfig{})
			kokoro.LoadModel(&domain.Model{ID: "kokoro-v1", Type: domain.ModelTypeTTS, Format: domain.ModelFormatONNX})
			kokoro.LoadModel(&domain.Model{ID: "kokoro-de", Type: domain.ModelTypeTTS, Format: domain.ModelFormatONNX})

			piper := inference.NewONNXBackend()
			piper.Initialize(backendConfig)
			piper.LoadModel(&domain.Model{ID: "piper-de_DE", Type: domain.ModelTypeTTS, Format: domain.ModelFormatONNX})

			orpheus := inference.NewGGMLBackend()
			orpheus.Initialize(backendConfig)
			orpheus.LoadModel(&domain.Model{ID: "orpheus-de", Type: domain.ModelTypeTTS, Format: domain.ModelFormatGGUF})
			orpheus.LoadModel(&domain.Model{ID: "orpheus-en", Type: domain.ModelTypeTTS, Format: domain.ModelFormatGGUF})

			whisper := inference.NewWhisperBackend()
			whisper.Initialize(backendConfig)
			whisper.LoadModel(&domain.Model{ID: "whisper-turbo", Type: domain.ModelTypeSTT, Format: domain.ModelFormatBin})
			whisper.LoadModel(&domain.Model{ID: "whisper-large-v3", Type: domain.ModelTypeSTT, Format: domain.ModelFormatBin})

			srv := api.NewServer(cfg)
			srv.RegisterBackend("kokoro-v1", kokoro)
			srv.RegisterBackend("kokoro-de", kokoro)
			srv.RegisterBackend("piper-de_DE", piper)
			srv.RegisterBackend("orpheus-de", orpheus)
			srv.RegisterBackend("orpheus-en", orpheus)
			srv.RegisterBackend("whisper-turbo", whisper)
			srv.RegisterBackend("whisper-large-v3", whisper)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-sigCh
				log.Println("Shutting down...")
				srv.Shutdown(ctx)
				cancel()
			}()

			addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
			log.Printf("Aurora Audio Server v1.0.0")
			log.Printf("CPU: %.0f compute score", cpuBackend.ComputeScore())
			log.Printf("API: http://%s", addr)
			log.Printf("Models: kokoro-v1, kokoro-de, piper-de_DE, orpheus-de, orpheus-en, whisper-turbo, whisper-large-v3")

			return srv.Start()
		},
	}

	cmd.Flags().StringVar(&host, "host", "", "Bind host (default: 0.0.0.0)")
	cmd.Flags().IntVar(&port, "port", 0, "Bind port (default: 11435)")
	cmd.Flags().BoolVar(&cpuOnly, "cpu-only", false, "Disable GPU backend")
	cmd.Flags().IntVar(&cpuThreads, "cpu-threads", 0, "CPU threads (0 = auto)")
	cmd.Flags().StringVar(&configPath, "config", "", "Path to config file")

	return cmd
}
