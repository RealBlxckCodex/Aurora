package hardware

import (
	"context"
	"runtime"
	"sync"

	"github.com/RealBlxckCodex/Aurora/pkg/domain"
)

type LoadTracker struct {
	mu       sync.RWMutex
	current  float64
	models   map[string]*domain.Model
}

func NewLoadTracker() *LoadTracker {
	return &LoadTracker{
		models: make(map[string]*domain.Model),
	}
}

func (lt *LoadTracker) Add(modelID string) {
	lt.mu.Lock()
	defer lt.mu.Unlock()
	lt.current = min(lt.current+0.1, 1.0)
}

func (lt *LoadTracker) Remove(modelID string) {
	lt.mu.Lock()
	defer lt.mu.Unlock()
	delete(lt.models, modelID)
	lt.current = max(lt.current-0.1, 0)
}

func (lt *LoadTracker) Load() float64 {
	lt.mu.RLock()
	defer lt.mu.RUnlock()
	return lt.current
}

type ThreadPool struct {
	numWorkers int
}

func NewThreadPool(n int) *ThreadPool {
	return &ThreadPool{numWorkers: n}
}

type CPUBackend struct {
	id           string
	numCores     int
	numThreads   int
	totalRAM     uint64
	hasAVX       bool
	hasAVX2      bool
	hasAVX512    bool
	hasAVX512VNNI bool
	hasAMX       bool
	hasFMA       bool
	threadPool   *ThreadPool
	loadTracker  *LoadTracker
	loadedModels map[string]*domain.Model
	mu           sync.RWMutex
}

func NewCPUBackend() (*CPUBackend, error) {
	cpu := &CPUBackend{
		id:           "cpu",
		numCores:     runtime.NumCPU(),
		numThreads:   runtime.NumCPU(),
		totalRAM:     getTotalSystemMemory(),
		threadPool:   NewThreadPool(runtime.NumCPU()),
		loadTracker:  NewLoadTracker(),
		loadedModels: make(map[string]*domain.Model),
	}
	cpu.detectFeatures()
	return cpu, nil
}

func (c *CPUBackend) ID() string                          { return c.id }
func (c *CPUBackend) Type() BackendType                   { return BackendTypeCPU }
func (c *CPUBackend) IsAvailable() bool                    { return true }
func (c *CPUBackend) TotalMemory() uint64                  { return c.totalRAM }
func (c *CPUBackend) ComputeScore() float64 {
	score := float64(c.numCores)
	if c.hasAVX512 {
		score *= 2.0
	}
	if c.hasAMX {
		score *= 1.5
	}
	if c.hasAVX2 {
		score *= 1.3
	}
	return score
}
func (c *CPUBackend) FreeMemory() uint64 {
	return c.totalRAM / 2
}
func (c *CPUBackend) CurrentLoad() float64 {
	return c.loadTracker.Load()
}
func (c *CPUBackend) ActiveModels() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	ids := make([]string, 0, len(c.loadedModels))
	for id := range c.loadedModels {
		ids = append(ids, id)
	}
	return ids
}

func (c *CPUBackend) Execute(ctx context.Context, task ComputeTask) (Result, error) {
	return Result{Data: task.Input}, nil
}

func (c *CPUBackend) LoadModel(model *domain.Model) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.loadedModels[model.ID] = model
	c.loadTracker.Add(model.ID)
	return nil
}

func (c *CPUBackend) UnloadModel(modelID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.loadedModels, modelID)
	c.loadTracker.Remove(modelID)
	return nil
}

func (c *CPUBackend) detectFeatures() {
	c.hasAVX = true
	c.hasAVX2 = true
	c.hasFMA = true
}
