package tts

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

type PiperConfig struct {
	BinaryPath string
	ModelPath  string
	Voice      string
	Speed      float64
}

type Piper struct {
	config PiperConfig
}

func NewPiper(config PiperConfig) *Piper {
	return &Piper{config: config}
}

func (p *Piper) Synthesize(ctx context.Context, text string, outputPath string) error {
	args := []string{
		"--model", p.config.ModelPath,
		"--output", outputPath,
	}
	if p.config.Speed > 0 {
		args = append(args, "--length-scale", fmt.Sprintf("%.2f", 1.0/p.config.Speed))
	}

	cmd := exec.CommandContext(ctx, p.config.BinaryPath, args...)

	var stdin bytes.Buffer
	stdin.WriteString(text)
	cmd.Stdin = &stdin

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("piper failed: %w\nstderr: %s", err, stderr.String())
	}
	return nil
}
