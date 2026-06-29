package inference

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/RealBlxckCodex/Aurora/pkg/domain"
)

type GGMLBackend struct {
	*BaseBackend
	models      map[string]*domain.Model
	modelDir    string
	serverPool  *LlamaServerPool
}

func NewGGMLBackend() *GGMLBackend {
	return &GGMLBackend{
		BaseBackend: NewBaseBackend("ggml", BackendTypeGGML),
		models:      make(map[string]*domain.Model),
		serverPool:  NewLlamaServerPool(),
	}
}

func (g *GGMLBackend) Initialize(config BackendConfig) error {
	g.modelDir = config.ModelDir
	return nil
}

func (g *GGMLBackend) Shutdown() error {
	g.serverPool.StopAll()
	g.models = nil
	return nil
}

func (g *GGMLBackend) LoadModel(model *domain.Model) error {
	g.models[model.ID] = model
	g.loaded[model.ID] = model

	go func() {
		modelPath := fmt.Sprintf("%s/%s/model.gguf", g.modelDir, model.ID)
		if _, err := os.Stat(modelPath); err == nil {
			g.serverPool.Start(context.Background(), model.ID)
		}
	}()

	return nil
}

func (g *GGMLBackend) UnloadModel(modelID string) error {
	g.serverPool.Stop(modelID)
	delete(g.models, modelID)
	delete(g.loaded, modelID)
	return nil
}

func (g *GGMLBackend) Infer(ctx context.Context, request InferenceRequest) (InferenceResponse, error) {
	if _, ok := g.models[request.ModelID]; !ok {
		return InferenceResponse{}, fmt.Errorf("model %s not loaded", request.ModelID)
	}

	modelPath := fmt.Sprintf("%s/%s/model.gguf", g.modelDir, request.ModelID)
	if _, err := os.Stat(modelPath); err != nil {
		return InferenceResponse{}, fmt.Errorf("model not found at %s", modelPath)
	}

	prompt := g.buildPrompt(request)
	temperature := g.temperature(request)

	content, err := g.serverPool.Infer(ctx, request.ModelID, prompt, temperature)
	if err != nil {
		return InferenceResponse{}, fmt.Errorf("inference: %w", err)
	}

	return InferenceResponse{
		Text:      content,
		AudioData: []byte(content),
		Duration:  0,
	}, nil
}

func (g *GGMLBackend) buildPrompt(req InferenceRequest) string {
	emotionTag := ""
	if req.Voice != "" {
		parts := strings.SplitN(req.Voice, "/", 2)
		if len(parts) == 2 {
			emotionTag = parts[1]
		}
	}

	switch {
	case strings.Contains(req.ModelID, "orpheus"):
		if emotionTag != "" {
			return fmt.Sprintf("<|audio|>[%s]%s<|eos|>", emotionTag, req.Input)
		}
		return fmt.Sprintf("<|audio|>%s<|eos|>", req.Input)
	default:
		return req.Input
	}
}

func (g *GGMLBackend) temperature(req InferenceRequest) float64 {
	if strings.Contains(req.ModelID, "orpheus") {
		return 0.8
	}
	return 0.0
}

func (g *GGMLBackend) Health() BackendHealth {
	return BackendHealth{
		Status: "healthy",
		Models: len(g.models),
	}
}

func (g *GGMLBackend) Stats() BackendStats {
	return BackendStats{}
}
