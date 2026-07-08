//go:build !cgo

package inference

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/RealBlxckCodex/Aurora/pkg/domain"
)

type KokoroBackend struct {
	*BaseBackend
	models   map[string]*domain.Model
	modelDir string
}

func NewKokoroBackend() *KokoroBackend {
	return &KokoroBackend{
		BaseBackend: NewBaseBackend("kokoro", BackendTypeONNX),
		models:      make(map[string]*domain.Model),
	}
}

func (k *KokoroBackend) Initialize(config BackendConfig) error {
	k.modelDir = config.ModelDir
	return nil
}

func (k *KokoroBackend) Shutdown() error {
	k.models = nil
	return nil
}

func (k *KokoroBackend) LoadModel(model *domain.Model) error {
	k.models[model.ID] = model
	k.loaded[model.ID] = model
	return nil
}

func (k *KokoroBackend) UnloadModel(modelID string) error {
	delete(k.models, modelID)
	delete(k.loaded, modelID)
	return nil
}

func (k *KokoroBackend) Infer(ctx context.Context, req InferenceRequest) (InferenceResponse, error) {
	tmpFile := fmt.Sprintf("/tmp/aurora-espeak-%d.wav", time.Now().UnixNano())
	defer os.Remove(tmpFile)

	lang := "en"
	cmd := exec.CommandContext(ctx, "espeak-ng", "-w", tmpFile, "-v", lang, req.Input)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return InferenceResponse{}, fmt.Errorf("espeak fallback: %w", err)
	}

	audio, err := os.ReadFile(tmpFile)
	if err != nil {
		return InferenceResponse{}, fmt.Errorf("read audio: %w", err)
	}

	return InferenceResponse{
		AudioData: audio,
		Duration:  time.Second,
	}, nil
}

func (k *KokoroBackend) Health() BackendHealth {
	return BackendHealth{Status: "healthy", Models: len(k.models)}
}

func (k *KokoroBackend) Stats() BackendStats {
	return BackendStats{}
}
