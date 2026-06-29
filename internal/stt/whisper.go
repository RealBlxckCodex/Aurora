package stt

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

type WhisperConfig struct {
	BinaryPath string
	ModelPath  string
	Language   string
}

type Whisper struct {
	config WhisperConfig
}

func NewWhisper(config WhisperConfig) *Whisper {
	return &Whisper{config: config}
}

func (w *Whisper) Transcribe(ctx context.Context, audioPath string) (string, error) {
	args := []string{
		"--model", w.config.ModelPath,
		"--file", audioPath,
		"--output-txt",
	}

	if w.config.Language != "" {
		args = append(args, "--language", w.config.Language)
	}

	cmd := exec.CommandContext(ctx, w.config.BinaryPath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("whisper failed: %w\nstderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}
