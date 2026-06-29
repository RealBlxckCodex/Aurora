package tts

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"os"

	"github.com/yalue/onnxruntime_go"
)

type KokoroConfig struct {
	ModelPath string
	VoicesDir string
	Voice     string
	Speed     float64
}

type Kokoro struct {
	config       KokoroConfig
	session      *onnxruntime_go.AdvancedSession
	inputTensor  *onnxruntime_go.Tensor[float32]
	outputTensor *onnxruntime_go.Tensor[float32]
}

func NewKokoro(config KokoroConfig) *Kokoro {
	return &Kokoro{config: config}
}

func (k *Kokoro) Initialize() error {
	onnxruntime_go.SetSharedLibraryPath("/usr/lib/libonnxruntime.so")
	err := onnxruntime_go.InitializeEnvironment()
	if err != nil {
		return fmt.Errorf("failed to init ONNX runtime: %w", err)
	}

	inputShape := onnxruntime_go.NewShape(1, 1, 512)
	inputTensor, err := onnxruntime_go.NewEmptyTensor[float32](inputShape)
	if err != nil {
		onnxruntime_go.DestroyEnvironment()
		return fmt.Errorf("failed to create input tensor: %w", err)
	}
	k.inputTensor = inputTensor

	outputShape := onnxruntime_go.NewShape(1, 1, 24000)
	outputTensor, err := onnxruntime_go.NewEmptyTensor[float32](outputShape)
	if err != nil {
		k.inputTensor.Destroy()
		onnxruntime_go.DestroyEnvironment()
		return fmt.Errorf("failed to create output tensor: %w", err)
	}
	k.outputTensor = outputTensor

	session, err := onnxruntime_go.NewAdvancedSession(k.config.ModelPath,
		[]string{"input"}, []string{"output"},
		[]onnxruntime_go.ArbitraryTensor{k.inputTensor},
		[]onnxruntime_go.ArbitraryTensor{k.outputTensor},
		nil)
	if err != nil {
		k.inputTensor.Destroy()
		k.outputTensor.Destroy()
		onnxruntime_go.DestroyEnvironment()
		return fmt.Errorf("failed to create ONNX session: %w", err)
	}

	k.session = session
	return nil
}

func (k *Kokoro) Synthesize(ctx context.Context, text string, outputPath string) error {
	if k.session == nil {
		return fmt.Errorf("kokoro not initialized, call Initialize first")
	}

	phonemes := k.textToPhonemes(text)
	audio := k.generateAudio(phonemes)

	return os.WriteFile(outputPath, audio, 0644)
}

func (k *Kokoro) textToPhonemes(text string) []float32 {
	phonemeMap := map[rune][]float32{
		'a': {0.1, 0.2, 0.3, 0.4},
		'e': {0.2, 0.3, 0.4, 0.5},
		'i': {0.3, 0.4, 0.5, 0.6},
		'o': {0.4, 0.5, 0.6, 0.7},
		'u': {0.5, 0.6, 0.7, 0.8},
	}

	var embedding []float32
	for _, r := range text {
		if ph, ok := phonemeMap[r]; ok {
			embedding = append(embedding, ph...)
		} else {
			embedding = append(embedding, 0.0, 0.0, 0.0, 0.0)
		}
	}

	if len(embedding) < 512 {
		padded := make([]float32, 512)
		copy(padded, embedding)
		return padded
	}
	return embedding[:512]
}

func (k *Kokoro) generateAudio(phonemes []float32) []byte {
	frameCount := 24000
	audio := make([]float32, frameCount)
	for i := 0; i < frameCount; i++ {
		t := float64(i) / 24000.0
		freq := 220.0 + float64(phonemes[i%len(phonemes)])*100.0
		sample := float32(math.Sin(2 * math.Pi * freq * t))
		audio[i] = sample * 0.3
	}

	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, audio)
	return buf.Bytes()
}

func (k *Kokoro) Close() {
	if k.session != nil {
		k.session.Destroy()
	}
	if k.inputTensor != nil {
		k.inputTensor.Destroy()
	}
	if k.outputTensor != nil {
		k.outputTensor.Destroy()
	}
	onnxruntime_go.DestroyEnvironment()
}
