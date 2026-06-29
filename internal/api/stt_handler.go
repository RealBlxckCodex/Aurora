package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/RealBlxckCodex/Aurora/internal/inference"
)

type STTRequest struct {
	Model    string `json:"model"`
	Language string `json:"language"`
	Prompt   string `json:"prompt"`
}

type STTResponse struct {
	Text     string  `json:"text"`
	Language string  `json:"language"`
	Duration float64 `json:"duration"`
}

func (s *Server) handleSTT(w http.ResponseWriter, r *http.Request) {
	model := r.FormValue("model")
	language := r.FormValue("language")

	if model == "" {
		http.Error(w, `{"error":"model is required"}`, http.StatusBadRequest)
		return
	}

	backend, ok := s.backends[model]
	if !ok {
		http.Error(w, `{"error":"model not found"}`, http.StatusNotFound)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"file required: %s"}`, err.Error()), http.StatusBadRequest)
		return
	}
	defer file.Close()

	tmpFile := fmt.Sprintf("/tmp/aurora-stt-%d%s", time.Now().UnixNano(), extFromMIME(header.Header.Get("Content-Type")))
	f, err := os.Create(tmpFile)
	if err != nil {
		http.Error(w, `{"error":"cannot create temp file"}`, http.StatusInternalServerError)
		return
	}
	defer os.Remove(tmpFile)

	if _, err := io.Copy(f, file); err != nil {
		f.Close()
		http.Error(w, `{"error":"cannot save file"}`, http.StatusInternalServerError)
		return
	}
	f.Close()

	resp, err := backend.Infer(r.Context(), inference.InferenceRequest{
		ModelID:  model,
		Language: language,
		Input:    tmpFile,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"inference failed: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	result := STTResponse{
		Text:     resp.Text,
		Language: language,
		Duration: resp.Duration.Seconds(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func extFromMIME(mime string) string {
	switch mime {
	case "audio/wav", "audio/wave":
		return ".wav"
	case "audio/mpeg", "audio/mp3":
		return ".mp3"
	case "audio/flac":
		return ".flac"
	case "audio/ogg":
		return ".ogg"
	case "audio/webm":
		return ".webm"
	case "audio/mp4":
		return ".m4a"
	default:
		return ".wav"
	}
}
