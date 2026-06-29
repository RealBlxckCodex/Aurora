package api

import (
	"encoding/json"
	"net/http"

	"github.com/RealBlxckCodex/Aurora/internal/inference"
)

type ModelInfo struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	Backend string `json:"backend"`
	Loaded bool   `json:"loaded"`
}

type LanguageInfo struct {
	Code         string   `json:"code"`
	Name         string   `json:"name"`
	NativeName   string   `json:"native_name"`
	TTSModels    []string `json:"tts_models"`
	STTModels    []string `json:"stt_models"`
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	models := make([]ModelInfo, 0)
	for id, backend := range s.backends {
		models = append(models, ModelInfo{
			ID:      id,
			Backend: string(backend.Type()),
			Loaded:  true,
		})
	}

	status := map[string]interface{}{
		"version": "1.0.0",
		"models":  models,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (s *Server) handleListModels(w http.ResponseWriter, r *http.Request) {
	models := make([]ModelInfo, 0)
	for id, backend := range s.backends {
		modelType := "tts"
		if backend.Type() == inference.BackendTypeWhisper {
			modelType = "stt"
		}
		models = append(models, ModelInfo{
			ID:      id,
			Type:    modelType,
			Backend: string(backend.Type()),
			Loaded:  true,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"models": models,
	})
}

func (s *Server) handleListLanguages(w http.ResponseWriter, r *http.Request) {
	languages := []LanguageInfo{
		{Code: "de", Name: "German", NativeName: "Deutsch", TTSModels: []string{"kokoro-v1", "piper-de_DE", "orpheus-de"}, STTModels: []string{"whisper-large-v3", "whisper-turbo"}},
		{Code: "en", Name: "English", NativeName: "English", TTSModels: []string{"kokoro-v1", "orpheus-en"}, STTModels: []string{"whisper-large-v3", "whisper-turbo"}},
		{Code: "fr", Name: "French", NativeName: "Français", TTSModels: []string{"kokoro-v1"}, STTModels: []string{"whisper-large-v3"}},
		{Code: "es", Name: "Spanish", NativeName: "Español", TTSModels: []string{"kokoro-v1"}, STTModels: []string{"whisper-large-v3"}},
		{Code: "ja", Name: "Japanese", NativeName: "日本語", TTSModels: []string{"kokoro-v1"}, STTModels: []string{"whisper-large-v3"}},
		{Code: "ko", Name: "Korean", NativeName: "한국어", TTSModels: []string{"kokoro-v1"}, STTModels: []string{"whisper-large-v3"}},
		{Code: "zh", Name: "Chinese", NativeName: "中文", TTSModels: []string{"kokoro-v1"}, STTModels: []string{"whisper-large-v3"}},
		{Code: "it", Name: "Italian", NativeName: "Italiano", TTSModels: []string{"kokoro-v1"}, STTModels: []string{"whisper-large-v3"}},
		{Code: "pt", Name: "Portuguese", NativeName: "Português", TTSModels: []string{"kokoro-v1"}, STTModels: []string{"whisper-large-v3"}},
		{Code: "ru", Name: "Russian", NativeName: "Русский", TTSModels: []string{"kokoro-v1"}, STTModels: []string{"whisper-large-v3"}},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"languages": languages,
	})
}

func inferenceRequestFromTTS(req TTSRequest) inference.InferenceRequest {
	return inference.InferenceRequest{
		ModelID: req.Model,
		Input:   req.Input,
		Voice:   req.Voice,
		Speed:   req.Speed,
		Format:  req.ResponseFormat,
	}
}
