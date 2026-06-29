package inference

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/RealBlxckCodex/Aurora/internal/audio"
	"github.com/RealBlxckCodex/Aurora/pkg/domain"
	"github.com/yalue/onnxruntime_go"
)

type KokoroBackend struct {
	*BaseBackend
	models    map[string]*domain.Model
	modelDir  string
	session   *onnxruntime_go.DynamicAdvancedSession
	voices    map[string][]float32
	voiceList []string
	g2pLang   string
}

type voiceEntry struct {
	id   string
	data []float32
}

func NewKokoroBackend() *KokoroBackend {
	return &KokoroBackend{
		BaseBackend: NewBaseBackend("kokoro", BackendTypeONNX),
		models:      make(map[string]*domain.Model),
		voices:      make(map[string][]float32),
	}
}

func (k *KokoroBackend) Initialize(config BackendConfig) error {
	k.modelDir = config.ModelDir
	onnxruntime_go.SetSharedLibraryPath("/usr/local/onnxruntime/lib/libonnxruntime.so")
	return nil
}

func (k *KokoroBackend) Shutdown() error {
	if k.session != nil {
		k.session.Destroy()
	}
	onnxruntime_go.DestroyEnvironment()
	k.models = nil
	k.voices = nil
	return nil
}

func (k *KokoroBackend) LoadModel(model *domain.Model) error {
	k.models[model.ID] = model
	k.loaded[model.ID] = model

	modelDir := filepath.Join(k.modelDir, model.ID)
	onnxPath := filepath.Join(modelDir, "model.onnx")
	if _, err := os.Stat(onnxPath); err != nil {
		return fmt.Errorf("model.onnx not found in %s", modelDir)
	}

	voicesPath := filepath.Join(modelDir, "voices.bin")
	if _, err := os.Stat(voicesPath); err == nil {
		if err := k.loadVoices(voicesPath); err != nil {
			return fmt.Errorf("load voices: %w", err)
		}
	}

	opts, err := onnxruntime_go.NewSessionOptions()
	if err != nil {
		return fmt.Errorf("session options: %w", err)
	}
	defer opts.Destroy()

	session, err := onnxruntime_go.NewDynamicAdvancedSession(onnxPath,
		[]string{"input_ids", "style", "speed"},
		[]string{"waveform"},
		opts,
	)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	k.session = session
	return nil
}

func (k *KokoroBackend) UnloadModel(modelID string) error {
	delete(k.models, modelID)
	delete(k.loaded, modelID)
	return nil
}

func (k *KokoroBackend) Infer(ctx context.Context, req InferenceRequest) (InferenceResponse, error) {
	if _, ok := k.models[req.ModelID]; !ok {
		return InferenceResponse{}, fmt.Errorf("model %s not loaded", req.ModelID)
	}
	if k.session == nil {
		return k.fallbackSynthesis(ctx, req)
	}

	return k.onnxInfer(ctx, req)
}

func (k *KokoroBackend) onnxInfer(ctx context.Context, req InferenceRequest) (InferenceResponse, error) {
	voiceName := k.selectVoice(req.Voice)

	phonemes, err := k.phonemize(req.Input, voiceName)
	if err != nil {
		return InferenceResponse{}, fmt.Errorf("g2p: %w", err)
	}

	tokens := tokenizePhonemes(phonemes)
	if len(tokens) > 510 {
		tokens = tokens[:510]
	}

	padded := make([]int64, 0, len(tokens)+2)
	padded = append(padded, 0)
	padded = append(padded, tokens...)
	padded = append(padded, 0)

	voiceEmb, ok := k.voices[voiceName]
	if !ok {
		voiceEmb = k.pickFirstVoice()
	}
	if voiceEmb == nil {
		return InferenceResponse{}, fmt.Errorf("no voice embeddings available")
	}

	embLen := len(voiceEmb) / 256
	styleIdx := len(tokens)
	if styleIdx >= embLen {
		styleIdx = embLen - 1
	}
	if styleIdx < 0 {
		styleIdx = 0
	}
	styleData := voiceEmb[styleIdx*256 : (styleIdx+1)*256]

	speedVal := float32(req.Speed)
	if speedVal <= 0 {
		speedVal = 1.0
	}

	inputIDs, err := onnxruntime_go.NewTensor(
		onnxruntime_go.NewShape(1, int64(len(padded))), padded)
	if err != nil {
		return InferenceResponse{}, fmt.Errorf("input tensor: %w", err)
	}
	defer inputIDs.Destroy()

	styleTensor, err := onnxruntime_go.NewTensor(
		onnxruntime_go.NewShape(1, 256), styleData)
	if err != nil {
		return InferenceResponse{}, fmt.Errorf("style tensor: %w", err)
	}
	defer styleTensor.Destroy()

	speedTensor, err := onnxruntime_go.NewTensor(
		onnxruntime_go.NewShape(1), []float32{speedVal})
	if err != nil {
		return InferenceResponse{}, fmt.Errorf("speed tensor: %w", err)
	}
	defer speedTensor.Destroy()

	outputs := []onnxruntime_go.Value{nil}
	if err := k.session.Run(
		[]onnxruntime_go.Value{inputIDs, styleTensor, speedTensor},
		outputs,
	); err != nil {
		return InferenceResponse{}, fmt.Errorf("run: %w", err)
	}

	if outputs[0] != nil {
		defer outputs[0].Destroy()
	}

	outTensor, ok := outputs[0].(*onnxruntime_go.Tensor[float32])
	if !ok {
		return InferenceResponse{}, fmt.Errorf("unexpected output type")
	}

	audioData := outTensor.GetData()
	wavBuf := audio.EncodeWAV(audioData, 24000)

	return InferenceResponse{
		AudioData: wavBuf,
		Duration:  time.Duration(len(audioData)) * time.Second / 24000,
	}, nil
}

func (k *KokoroBackend) phonemize(text, voiceName string) (string, error) {
	lang := guessLanguage(voiceName)

	var out bytes.Buffer
	cmd := exec.Command("espeak-ng", "-q", "-v", lang, "--ipa", "-x", text)
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("espeak failed: %w", err)
	}

	return strings.TrimSpace(out.String()), nil
}

func (k *KokoroBackend) selectVoice(requestVoice string) string {
	if requestVoice != "" {
		parts := strings.SplitN(requestVoice, "/", 2)
		voiceID := parts[len(parts)-1]
		if _, ok := k.voices[voiceID]; ok {
			return voiceID
		}
	}

	if len(k.voiceList) > 0 {
		best := k.voiceList[0]
		for _, v := range k.voiceList {
			if strings.HasPrefix(v, "df_") {
				return v
			}
		}
		return best
	}

	return "af_heart"
}

func (k *KokoroBackend) pickFirstVoice() []float32 {
	for _, v := range k.voices {
		return v
	}
	return nil
}

func (k *KokoroBackend) loadVoices(path string) error {
	if strings.HasSuffix(path, ".npz") {
		return k.loadVoicesNPZ(path)
	}

	r, err := zip.OpenReader(path)
	if err != nil {
		return fmt.Errorf("open voices zip: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		id := strings.TrimSuffix(f.Name, ".npy")
		rc, err := f.Open()
		if err != nil {
			continue
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			continue
		}
		parsed, err := parseNpy(data)
		if err != nil {
			continue
		}
		k.voices[id] = parsed
		k.voiceList = append(k.voiceList, id)
	}

	return nil
}

func (k *KokoroBackend) loadVoicesNPZ(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}
	for _, f := range r.File {
		id := strings.TrimSuffix(f.Name, ".npy")
		rc, err := f.Open()
		if err != nil {
			continue
		}
		raw, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			continue
		}
		parsed, err := parseNpy(raw)
		if err != nil {
			continue
		}
		k.voices[id] = parsed
		k.voiceList = append(k.voiceList, id)
	}
	return nil
}

func (k *KokoroBackend) fallbackSynthesis(ctx context.Context, req InferenceRequest) (InferenceResponse, error) {
	tmpFile := fmt.Sprintf("/tmp/aurora-espeak-%d.wav", time.Now().UnixNano())
	defer os.Remove(tmpFile)

	cmd := exec.CommandContext(ctx, "espeak-ng", "-w", tmpFile, "-v", "de", req.Input)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return InferenceResponse{}, fmt.Errorf("espeak: %w\n%s", err, stderr.String())
	}

	audio, err := os.ReadFile(tmpFile)
	if err != nil {
		return InferenceResponse{}, fmt.Errorf("read: %w", err)
	}

	return InferenceResponse{
		AudioData: audio,
		Duration:  time.Second,
	}, nil
}

func (k *KokoroBackend) Health() BackendHealth {
	return BackendHealth{Status: "healthy", Models: len(k.models)}
}

func (k *KokoroBackend) Stats() BackendStats {
	return BackendStats{}
}

func guessLanguage(voiceID string) string {
	switch {
	case strings.HasPrefix(voiceID, "af_") || strings.HasPrefix(voiceID, "am_") ||
		strings.HasPrefix(voiceID, "bf_") || strings.HasPrefix(voiceID, "bm_") ||
		strings.HasPrefix(voiceID, "ef_") || strings.HasPrefix(voiceID, "em_") ||
		strings.HasPrefix(voiceID, "if_") || strings.HasPrefix(voiceID, "im_") ||
		strings.HasPrefix(voiceID, "pf_") || strings.HasPrefix(voiceID, "pm_"):
		return "en"
	case strings.HasPrefix(voiceID, "jf_") || strings.HasPrefix(voiceID, "jm_"):
		return "ja"
	case strings.HasPrefix(voiceID, "zf_") || strings.HasPrefix(voiceID, "zm_"):
		return "zh"
	case strings.HasPrefix(voiceID, "hf_") || strings.HasPrefix(voiceID, "hm_"):
		return "ko"
	case strings.HasPrefix(voiceID, "ff_") || strings.HasPrefix(voiceID, "fm_"):
		return "fr"
	case strings.HasPrefix(voiceID, "df_") || strings.HasPrefix(voiceID, "dm_"):
		return "de"
	}
	return "en"
}

func tokenizePhonemes(phonemes string) []int64 {
	tokens := make([]int64, 0, len(phonemes))
	for _, r := range phonemes {
		id := charToID(r)
		if id == 0 && r >= 'A' && r <= 'Z' {
			id = charToID(rune(r - 'A' + 'a'))
		}
		tokens = append(tokens, id)
	}
	return tokens
}

func charToID(r rune) int64 {
	switch r {
	case '$':
		return 0
	case ';':
		return 1
	case ':':
		return 2
	case ',':
		return 3
	case '.':
		return 4
	case '!':
		return 5
	case '?':
		return 6
	case '"':
		return 11
	case '(':
		return 12
	case ')':
		return 13
	case ' ':
		return 16
	case 'A':
		return 24
	case 'I':
		return 25
	case 'O':
		return 31
	case 'Q':
		return 33
	case 'S':
		return 35
	case 'T':
		return 36
	case 'W':
		return 39
	case 'Y':
		return 41
	case 'a':
		return 43
	case 'b':
		return 44
	case 'c':
		return 45
	case 'd':
		return 46
	case 'e':
		return 47
	case 'f':
		return 48
	case 'h':
		return 50
	case 'i':
		return 51
	case 'j':
		return 52
	case 'k':
		return 53
	case 'l':
		return 54
	case 'm':
		return 55
	case 'n':
		return 56
	case 'o':
		return 57
	case 'p':
		return 58
	case 'q':
		return 59
	case 'r':
		return 60
	case 's':
		return 61
	case 't':
		return 62
	case 'u':
		return 63
	case 'v':
		return 64
	case 'w':
		return 65
	case 'x':
		return 66
	case 'y':
		return 67
	case 'z':
		return 68
	case 'ɡ':
		return 92
	case 'ŋ':
		return 112
	case 'ʃ':
		return 131
	case 'ʒ':
		return 147
	case 'ə':
		return 83
	case 'ɛ':
		return 86
	case 'ɪ':
		return 102
	case 'ɔ':
		return 76
	case 'ʊ':
		return 135
	case 'ɑ':
		return 69
	case 'ʌ':
		return 138
	case 'æ':
		return 72
	case 'œ':
		return 120
	case 'ø':
		return 116
	case 'ð':
		return 81
	case 'θ':
		return 119
	case 'β':
		return 75
	case 'χ':
		return 142
	case 'ɣ':
		return 139
	case 'ʔ':
		return 148
	case 'ˈ':
		return 156
	case 'ˌ':
		return 157
	case 'ː':
		return 158
	case 'ʰ':
		return 162
	case 'ʲ':
		return 164
	case 'ɐ':
		return 70
	case 'ɒ':
		return 71
	case 'ɕ':
		return 77
	case 'ç':
		return 78
	case 'ɖ':
		return 80
	case 'ɚ':
		return 85
	case 'ɜ':
		return 87
	case 'ɟ':
		return 90
	case 'ɥ':
		return 99
	case 'ɨ':
		return 101
	case 'ɯ':
		return 110
	case 'ɰ':
		return 111
	case 'ɳ':
		return 113
	case 'ɲ':
		return 114
	case 'ɴ':
		return 115
	case 'ɸ':
		return 118
	case 'ɹ':
		return 123
	case 'ɾ':
		return 125
	case 'ɻ':
		return 126
	case 'ʁ':
		return 128
	case 'ɽ':
		return 129
	case 'ʂ':
		return 130
	case 'ʈ':
		return 132
	case 'ʋ':
		return 136
	case 'ʎ':
		return 143
	case 'ʣ':
		return 18
	case 'ʤ':
		return 82
	case 'ʥ':
		return 19
	case 'ʦ':
		return 20
	case 'ʧ':
		return 133
	case 'ʨ':
		return 21
	case 'ʝ':
		return 103
	case 'ɤ':
		return 140
	case 'ᵊ':
		return 42
	case 'ᵝ':
		return 22
	case 'ᵻ':
		return 177
	case 'ꭧ':
		return 23
	case '—':
		return 9
	case '…':
		return 10
	case '\u201C':
		return 14
	case '\u201D':
		return 15
	case '̃':
		return 17
	case '↓':
		return 169
	case '→':
		return 171
	case '↗':
		return 172
	case '↘':
		return 173
	}
	return 0
}

func parseNpy(data []byte) ([]float32, error) {
	if len(data) < 80 || string(data[:6]) != "\x93NUMPY" {
		return nil, fmt.Errorf("not a npy file")
	}
	headerLen := int(binary.LittleEndian.Uint16(data[8:10]))
	body := data[10+headerLen:]
	bodyWords := len(body) / 4
	result := make([]float32, bodyWords)
	for i := range result {
		result[i] = math.Float32frombits(binary.LittleEndian.Uint32(body[i*4:]))
	}
	return result, nil
}
