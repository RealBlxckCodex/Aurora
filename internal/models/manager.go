package models

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/RealBlxckCodex/Aurora/internal/config"
	"github.com/RealBlxckCodex/Aurora/pkg/domain"
)

type Manager struct {
	store      *Store
	downloader *Downloader
	cfg        *config.ModelsConfig
}

func NewManager(cfg *config.ModelsConfig) *Manager {
	return &Manager{
		store:      NewStore(cfg.Dir),
		downloader: NewDownloader(),
		cfg:        cfg,
	}
}

func (m *Manager) Initialize() error {
	if err := m.store.Load(); err != nil {
		return err
	}
	if len(m.store.List()) == 0 {
		m.Scan()
	}
	return nil
}

func (m *Manager) Scan() {
	entries, err := os.ReadDir(m.cfg.Dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		id := entry.Name()
		if m.store.Get(id) != nil {
			continue
		}
		dir := filepath.Join(m.cfg.Dir, id)
		files, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			ext := filepath.Ext(f.Name())
			if ext == ".onnx" || ext == ".gguf" || ext == ".bin" {
				m.store.Add(&domain.Model{
					ID:        id,
					Installed: true,
				})
				break
			}
		}
	}
}

func (m *Manager) List() []*domain.Model {
	return m.store.List()
}

func (m *Manager) Get(id string) *domain.Model {
	return m.store.Get(id)
}

func (m *Manager) Install(ctx context.Context, modelID string) error {
	existing := m.store.Get(modelID)
	if existing != nil && existing.Installed {
		return fmt.Errorf("model %s already installed", modelID)
	}

	entry, ok := m.cfg.Entries[modelID]
	if !ok {
		return fmt.Errorf("model %s not found in config", modelID)
	}

	model := &domain.Model{
		ID:        modelID,
		URL:       entry.URL,
		SHA256:    entry.SHA256,
		Installed: false,
	}

	dest := filepath.Join(m.cfg.Dir, modelID, "model.bin")
	log.Printf("Downloading %s from %s...", modelID, entry.URL)

	if err := m.downloader.Download(ctx, entry.URL, dest, entry.SHA256); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	model.Installed = true
	if err := m.store.Add(model); err != nil {
		return fmt.Errorf("save model metadata: %w", err)
	}

	log.Printf("Model %s installed successfully", modelID)
	return nil
}

func (m *Manager) Remove(modelID string) error {
	model := m.store.Get(modelID)
	if model == nil {
		return fmt.Errorf("model %s not found", modelID)
	}

	if err := m.store.Remove(modelID); err != nil {
		return fmt.Errorf("remove from store: %w", err)
	}

	log.Printf("Model %s removed", modelID)
	return nil
}
