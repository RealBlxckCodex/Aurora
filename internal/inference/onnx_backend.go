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

type ONNXBackend struct {
	*BaseBackend
	models   map[string]*domain.Model
	modelDir string
	piperBin string
}

func NewONNXBackend() *ONNXBackend {
	return &ONNXBackend{
		BaseBackend: NewBaseBackend("onnx", BackendTypeONNX),
		models:      make(map[string]*domain.Model),
	}
}

func (o *ONNXBackend) Initialize(config BackendConfig) error {
	o.modelDir = config.ModelDir
	o.piperBin = "piper"
	return nil
}

func (o *ONNXBackend) Shutdown() error {
	o.models = nil
	return nil
}

func (o *ONNXBackend) LoadModel(model *domain.Model) error {
	o.models[model.ID] = model
	o.loaded[model.ID] = model
	return nil
}

func (o *ONNXBackend) UnloadModel(modelID string) error {
	delete(o.models, modelID)
	delete(o.loaded, modelID)
	return nil
}

func (o *ONNXBackend) Infer(ctx context.Context, request InferenceRequest) (InferenceResponse, error) {
	_, ok := o.models[request.ModelID]
	if !ok {
		return InferenceResponse{}, fmt.Errorf("model %s not loaded", request.ModelID)
	}

	modelPath := fmt.Sprintf("%s/%s/model.onnx", o.modelDir, request.ModelID)
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		modelPath = request.ModelID + ".onnx"
	}

	tmpFile := fmt.Sprintf("/tmp/aurora-piper-%d.wav", time.Now().UnixNano())
	defer os.Remove(tmpFile)

	args := []string{
		"-m", modelPath,
		"-f", tmpFile,
	}
	if request.Speed > 0 && request.Speed != 1.0 {
		args = append(args, "--length-scale", fmt.Sprintf("%.2f", 1.0/request.Speed))
	}

	cmd := exec.CommandContext(ctx, o.piperBin, args...)
	var stdin bytes.Buffer
	stdin.WriteString(request.Input)
	cmd.Stdin = &stdin

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return InferenceResponse{}, fmt.Errorf("piper failed: %w\nstderr: %s", err, stderr.String())
	}

	audio, err := os.ReadFile(tmpFile)
	if err != nil {
		return InferenceResponse{}, fmt.Errorf("read output: %w", err)
	}

	return InferenceResponse{
		AudioData: audio,
		Duration:  100 * time.Millisecond,
	}, nil
}

func (o *ONNXBackend) Health() BackendHealth {
	return BackendHealth{
		Status: "healthy",
		Models: len(o.models),
	}
}

func (o *ONNXBackend) Stats() BackendStats {
	return BackendStats{
		TotalInferences: 0,
		AvgDuration:     0,
		TotalErrors:     0,
	}
}
