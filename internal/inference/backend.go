package inference

import (
	"context"
	"time"

	"github.com/RealBlxckCodex/Aurora/pkg/domain"
)

type BackendType string

const (
	BackendTypeGGML   BackendType = "ggml"
	BackendTypeONNX   BackendType = "onnx"
	BackendTypeWhisper BackendType = "whisper"
)

type InferenceRequest struct {
	ModelID  string
	Input    string
	Voice    string
	Language string
	Speed    float64
	Format   string
}

type InferenceResponse struct {
	AudioData []byte
	Text      string
	Duration  time.Duration
}

type StreamChunk struct {
	Data      []byte
	IsFinal   bool
	Timestamp time.Duration
}

type BackendHealth struct {
	Status    string `json:"status"`
	Uptime    string `json:"uptime"`
	Models    int    `json:"models_loaded"`
}

type BackendStats struct {
	TotalInferences int64   `json:"total_inferences"`
	AvgDuration     float64 `json:"avg_duration_ms"`
	TotalErrors     int64   `json:"total_errors"`
}

type IInferenceBackend interface {
	ID() string
	Type() BackendType
	Initialize(config BackendConfig) error
	Shutdown() error
	LoadModel(model *domain.Model) error
	UnloadModel(modelID string) error
	IsModelLoaded(modelID string) bool
	Infer(ctx context.Context, request InferenceRequest) (InferenceResponse, error)
	SupportsStreaming() bool
	Stream(ctx context.Context, request InferenceRequest) (<-chan StreamChunk, error)
	Health() BackendHealth
	Stats() BackendStats
}

type BackendConfig struct {
	ModelDir  string
	Threads   int
	GPUID     int
	UseGPU    bool
}

type BaseBackend struct {
	id       string
	bType    BackendType
	config   BackendConfig
	loaded   map[string]*domain.Model
}

func NewBaseBackend(id string, bType BackendType) *BaseBackend {
	return &BaseBackend{
		id:     id,
		bType:  bType,
		loaded: make(map[string]*domain.Model),
	}
}

func (b *BaseBackend) ID() string                           { return b.id }
func (b *BaseBackend) Type() BackendType                    { return b.bType }
func (b *BaseBackend) IsModelLoaded(modelID string) bool {
	_, ok := b.loaded[modelID]
	return ok
}
func (b *BaseBackend) SupportsStreaming() bool              { return false }
func (b *BaseBackend) Stream(ctx context.Context, req InferenceRequest) (<-chan StreamChunk, error) {
	return nil, nil
}
