package inference

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

type LlamaServerPool struct {
	mu          sync.Mutex
	servers     map[string]*llamaServerInstance
	llamaServer string
	modelDir    string
}

type llamaServerInstance struct {
	cmd     *exec.Cmd
	port    int
	modelID string
	ready   bool
	cancel  context.CancelFunc
}

type completionRequest struct {
	Prompt      string  `json:"prompt"`
	Temperature float64 `json:"temperature"`
	TopP        float64 `json:"top_p"`
	MaxTokens   int     `json:"n_predict"`
	Stream      bool    `json:"stream"`
}

type completionResponse struct {
	Content string `json:"content"`
}

func NewLlamaServerPool() *LlamaServerPool {
	return &LlamaServerPool{
		servers:     make(map[string]*llamaServerInstance),
		llamaServer: "/home/Workspace/Aurora/ext/llama.cpp/build/bin/llama-server",
		modelDir:    "/var/aurora/models",
	}
}

func (p *LlamaServerPool) Start(ctx context.Context, modelID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.servers[modelID]; ok {
		return nil
	}

	modelPath := filepath.Join(p.modelDir, modelID, "model.gguf")
	if _, err := os.Stat(modelPath); err != nil {
		return fmt.Errorf("model not found: %s", modelPath)
	}

	port := p.findFreePort()

	ctx, cancel := context.WithCancel(ctx)

	args := []string{
		"-m", modelPath,
		"--port", fmt.Sprintf("%d", port),
		"-t", "4",
		"-c", "2048",
		"--no-display-prompt",
		"--mlock",
	}

	cmd := exec.CommandContext(ctx, p.llamaServer, args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("start llama-server: %w", err)
	}

	inst := &llamaServerInstance{
		cmd:     cmd,
		port:    port,
		modelID: modelID,
		cancel:  cancel,
	}
	p.servers[modelID] = inst

	go func() {
		cmd.Wait()
	}()

	go func() {
		for i := 0; i < 30; i++ {
			if p.healthCheck(port) {
				p.mu.Lock()
				inst.ready = true
				p.mu.Unlock()
				return
			}
			time.Sleep(500 * time.Millisecond)
		}
		fmt.Printf("llama-server for %s failed to start: %s\n", modelID, stderr.String())
	}()

	return nil
}

func (p *LlamaServerPool) Infer(ctx context.Context, modelID, prompt string, temperature float64) (string, error) {
	inst, err := p.getInstance(modelID)
	if err != nil {
		return "", err
	}

	if !inst.ready {
		return "", fmt.Errorf("server for %s not ready yet", modelID)
	}

	req := completionRequest{
		Prompt:      prompt,
		Temperature: temperature,
		TopP:        0.95,
		MaxTokens:   2048,
		Stream:      false,
	}

	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf("http://127.0.0.1:%d/completion", inst.port),
		bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("http call: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var cr completionResponse
	if err := json.Unmarshal(respBody, &cr); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	return cr.Content, nil
}

func (p *LlamaServerPool) Stop(modelID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if inst, ok := p.servers[modelID]; ok {
		inst.cancel()
		inst.cmd.Process.Kill()
		delete(p.servers, modelID)
	}
}

func (p *LlamaServerPool) StopAll() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for id, inst := range p.servers {
		inst.cancel()
		inst.cmd.Process.Kill()
		delete(p.servers, id)
	}
}

func (p *LlamaServerPool) getInstance(modelID string) (*llamaServerInstance, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	inst, ok := p.servers[modelID]
	if !ok {
		return nil, fmt.Errorf("server for %s not started", modelID)
	}
	return inst, nil
}

func (p *LlamaServerPool) healthCheck(port int) bool {
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/health", port))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

func (p *LlamaServerPool) findFreePort() int {
	for port := 18000; port < 19000; port++ {
		if !p.portInUse(port) {
			return port
		}
	}
	return 18000
}

func (p *LlamaServerPool) portInUse(port int) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, inst := range p.servers {
		if inst.port == port {
			return true
		}
	}
	return false
}

func (p *LlamaServerPool) RunningServers() []string {
	p.mu.Lock()
	defer p.mu.Unlock()

	var ids []string
	for id, inst := range p.servers {
		if inst.ready {
			ids = append(ids, id)
		}
	}
	return ids
}
