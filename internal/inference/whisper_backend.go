package inference

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/RealBlxckCodex/Aurora/pkg/domain"
)

type WhisperBackend struct {
	*BaseBackend
	models    map[string]*domain.Model
	modelDir  string
	whisperBin string
	threads  int
}

func NewWhisperBackend() *WhisperBackend {
	return &WhisperBackend{
		BaseBackend: NewBaseBackend("whisper", BackendTypeWhisper),
		models:      make(map[string]*domain.Model),
	}
}

func (w *WhisperBackend) Initialize(config BackendConfig) error {
	w.modelDir = config.ModelDir
	w.threads = config.Threads

	candidates := []string{
		"/home/Workspace/Aurora/ext/whisper.cpp/build/bin/whisper-cli",
		"whisper-cli",
		"whisper.cpp",
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			w.whisperBin = c
			break
		}
		if path, err := exec.LookPath(c); err == nil {
			w.whisperBin = path
			break
		}
	}
	if w.whisperBin == "" {
		w.whisperBin = "/home/Workspace/Aurora/ext/whisper.cpp/build/bin/whisper-cli"
	}

	return nil
}

func (w *WhisperBackend) Shutdown() error {
	w.models = nil
	return nil
}

func (w *WhisperBackend) LoadModel(model *domain.Model) error {
	w.models[model.ID] = model
	w.loaded[model.ID] = model
	return nil
}

func (w *WhisperBackend) UnloadModel(modelID string) error {
	delete(w.models, modelID)
	delete(w.loaded, modelID)
	return nil
}

func (w *WhisperBackend) Infer(ctx context.Context, request InferenceRequest) (InferenceResponse, error) {
	if _, ok := w.models[request.ModelID]; !ok {
		return InferenceResponse{}, fmt.Errorf("model %s not loaded", request.ModelID)
	}

	inputFile := request.Input
	modelPath := filepath.Join(w.modelDir, request.ModelID, "model.bin")
	if _, err := os.Stat(modelPath); err != nil {
		return InferenceResponse{}, fmt.Errorf("model file not found: %s", modelPath)
	}

	tmpOutput := fmt.Sprintf("/tmp/aurora-whisper-%d", time.Now().UnixNano())
	defer cleanupWhisperOutput(tmpOutput)

	args := []string{
		"-m", modelPath,
		"-f", inputFile,
		"-otxt",
		"--output-file", tmpOutput,
	}
	if request.Language != "" {
		args = append(args, "-l", request.Language)
	}
	numThreads := w.threads
	if numThreads <= 0 {
		numThreads = runtime.NumCPU()
	}
	args = append(args, "-t", fmt.Sprintf("%d", numThreads), "-p", "4", "--no-prints")

	cmd := exec.CommandContext(ctx, w.whisperBin, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return InferenceResponse{}, fmt.Errorf("whisper failed: %w\n%s", err, stderr.String())
	}

	textFile := tmpOutput + ".txt"
	text := ""
	if data, err := os.ReadFile(textFile); err == nil {
		text = strings.TrimSpace(string(data))
	}

	return InferenceResponse{
		Text:     text,
		Duration: 500 * time.Millisecond,
	}, nil
}

func (w *WhisperBackend) Health() BackendHealth {
	return BackendHealth{Status: "healthy", Models: len(w.models)}
}

func (w *WhisperBackend) Stats() BackendStats {
	return BackendStats{}
}

func cleanupWhisperOutput(base string) {
	for _, ext := range []string{".txt", ".srt", ".vtt", ".tsv", ".json"} {
		os.Remove(base + ext)
	}
}
