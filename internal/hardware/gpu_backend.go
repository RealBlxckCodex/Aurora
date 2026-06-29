package hardware

import (
	"context"
	"sync"

	"github.com/RealBlxckCodex/Aurora/pkg/domain"
)

type GPUBackend struct {
	id           string
	deviceID     int
	totalVRAM    uint64
	freeVRAM     uint64
	computeScore float64
	loadTracker  *LoadTracker
	loadedModels map[string]*domain.Model
	mu           sync.RWMutex
	available    bool
}

func NewGPUBackend() *GPUBackend {
	return &GPUBackend{
		id:           "gpu:0",
		deviceID:     0,
		totalVRAM:    6 * 1024 * 1024 * 1024,
		freeVRAM:     6 * 1024 * 1024 * 1024,
		computeScore: 100.0,
		loadTracker:  NewLoadTracker(),
		loadedModels: make(map[string]*domain.Model),
		available:    false,
	}
}

func (g *GPUBackend) ID() string               { return g.id }
func (g *GPUBackend) Type() BackendType         { return BackendTypeGPU }
func (g *GPUBackend) IsAvailable() bool          { return g.available }
func (g *GPUBackend) TotalMemory() uint64        { return g.totalVRAM }
func (g *GPUBackend) FreeMemory() uint64         { return g.freeVRAM }
func (g *GPUBackend) ComputeScore() float64      { return g.computeScore }
func (g *GPUBackend) CurrentLoad() float64       { return g.loadTracker.Load() }
func (g *GPUBackend) ActiveModels() []string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	ids := make([]string, 0, len(g.loadedModels))
	for id := range g.loadedModels {
		ids = append(ids, id)
	}
	return ids
}

func (g *GPUBackend) Execute(ctx context.Context, task ComputeTask) (Result, error) {
	return Result{Data: task.Input}, nil
}

func (g *GPUBackend) LoadModel(model *domain.Model) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.loadedModels[model.ID] = model
	g.loadTracker.Add(model.ID)
	return nil
}

func (g *GPUBackend) UnloadModel(modelID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.loadedModels, modelID)
	g.loadTracker.Remove(modelID)
	return nil
}
