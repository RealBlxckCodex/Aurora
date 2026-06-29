package hardware

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type RoutingStrategy string

const (
	RoutingStrategyAuto   RoutingStrategy = "auto"
	RoutingStrategyCPU    RoutingStrategy = "cpu"
	RoutingStrategyGPU    RoutingStrategy = "gpu"
)

type SchedulerConfig struct {
	Strategy    RoutingStrategy
	CPUBoost    bool
}

type ResourceMonitor struct {
	cpuLoad float64
	gpuLoad float64
	mu      sync.RWMutex
}

func NewResourceMonitor() *ResourceMonitor {
	return &ResourceMonitor{}
}

func (rm *ResourceMonitor) CPULoad() float64 {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.cpuLoad
}

type HybridScheduler struct {
	backends []IComputeBackend
	strategy RoutingStrategy
	queues   map[string]chan ComputeTask
	monitor  *ResourceMonitor
	config   SchedulerConfig
	mu       sync.RWMutex
}

func NewHybridScheduler(backends []IComputeBackend, config SchedulerConfig) *HybridScheduler {
	return &HybridScheduler{
		backends: backends,
		strategy: config.Strategy,
		queues:   make(map[string]chan ComputeTask),
		monitor:  NewResourceMonitor(),
		config:   config,
	}
}

func (s *HybridScheduler) RouteTask(ctx context.Context, task ComputeTask) (Result, error) {
	backend := s.selectBackend(task)
	if backend == nil {
		return Result{}, fmt.Errorf("no suitable backend available")
	}
	return backend.Execute(ctx, task)
}

func (s *HybridScheduler) selectBackend(task ComputeTask) IComputeBackend {
	candidates := s.filterCandidates(task)
	if len(candidates) == 0 {
		return nil
	}

	if task.EstimatedDuration < 100*time.Millisecond && s.hasCPU(candidates) {
		cpu := s.getCPU(candidates)
		if cpu.CurrentLoad() < 0.8 {
			return cpu
		}
	}

	return s.selectLowestLoad(s.getGPUs(candidates))
}

func (s *HybridScheduler) filterCandidates(task ComputeTask) []IComputeBackend {
	var candidates []IComputeBackend
	for _, b := range s.backends {
		if b.IsAvailable() {
			candidates = append(candidates, b)
		}
	}
	return candidates
}

func (s *HybridScheduler) hasCPU(backends []IComputeBackend) bool {
	for _, b := range backends {
		if b.Type() == BackendTypeCPU {
			return true
		}
	}
	return false
}

func (s *HybridScheduler) getCPU(backends []IComputeBackend) IComputeBackend {
	for _, b := range backends {
		if b.Type() == BackendTypeCPU {
			return b
		}
	}
	return nil
}

func (s *HybridScheduler) getGPUs(backends []IComputeBackend) []IComputeBackend {
	var gpus []IComputeBackend
	for _, b := range backends {
		if b.Type() == BackendTypeGPU {
			gpus = append(gpus, b)
		}
	}
	return gpus
}

func (s *HybridScheduler) selectLowestLoad(backends []IComputeBackend) IComputeBackend {
	if len(backends) == 0 {
		return nil
	}
	best := backends[0]
	for _, b := range backends[1:] {
		if b.CurrentLoad() < best.CurrentLoad() {
			best = b
		}
	}
	return best
}
