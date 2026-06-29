package hardware

import (
	"context"
	"fmt"
	"time"

	"github.com/RealBlxckCodex/Aurora/pkg/domain"
)

type BackendType string

const (
	BackendTypeCPU BackendType = "cpu"
	BackendTypeGPU BackendType = "gpu"
)

type ComputeTask struct {
	ID               string
	ModelID          string
	Input            interface{}
	EstimatedDuration time.Duration
}

type Result struct {
	Data     interface{}
	Duration time.Duration
	Error    error
}

type IComputeBackend interface {
	ID() string
	Type() BackendType
	IsAvailable() bool
	TotalMemory() uint64
	FreeMemory() uint64
	ComputeScore() float64
	Execute(ctx context.Context, task ComputeTask) (Result, error)
	LoadModel(model *domain.Model) error
	UnloadModel(modelID string) error
	CurrentLoad() float64
	ActiveModels() []string
}

func fmtSize(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
