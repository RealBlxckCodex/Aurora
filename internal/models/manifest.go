package models

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

type FileEntry struct {
	URL      string `json:"url"`
	SHA256   string `json:"sha256"`
	Size     int64  `json:"size"`
}

type ModelEntry struct {
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Type        string                `json:"type"`
	Format      string                `json:"format"`
	Version     string                `json:"version"`
	Files       map[string]FileEntry  `json:"files"`
	Voices      []string              `json:"voices,omitempty"`
	Languages   []string              `json:"languages"`
}

type Manifest struct {
	SchemaVersion string                 `json:"schema_version"`
	BaseURL       string                 `json:"base_url"`
	Models        map[string]ModelEntry  `json:"models"`
}

func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}

	return &m, nil
}

func FetchManifest(ctx context.Context, registryURL string) (*Manifest, error) {
	manifestURL := registryURL + "/api/v1/manifest.json"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, manifestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status fetching manifest: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var m Manifest
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}

	return &m, nil
}

func DefaultManifestPath() string {
	exe, err := os.Executable()
	if err != nil {
		return "models/manifest.json"
	}
	return filepath.Join(filepath.Dir(exe), "..", "models", "manifest.json")
}

func (m *Manifest) GetModel(id string) (*ModelEntry, bool) {
	entry, ok := m.Models[id]
	if !ok {
		return nil, false
	}
	return &entry, true
}

func (m *Manifest) ListModels() []string {
	ids := make([]string, 0, len(m.Models))
	for id := range m.Models {
		ids = append(ids, id)
	}
	return ids
}
