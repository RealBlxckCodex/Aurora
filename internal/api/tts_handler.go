package api

import (
	"encoding/json"
	"net/http"
)

type TTSRequest struct {
	Model          string  `json:"model"`
	Input          string  `json:"input"`
	Voice          string  `json:"voice"`
	ResponseFormat string  `json:"response_format"`
	Speed          float64 `json:"speed"`
}

type TTSResponse struct {
	Text string `json:"text,omitempty"`
}

func (s *Server) handleTTS(w http.ResponseWriter, r *http.Request) {
	var req TTSRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Model == "" {
		http.Error(w, `{"error":"model is required"}`, http.StatusBadRequest)
		return
	}

	if req.Input == "" {
		http.Error(w, `{"error":"input is required"}`, http.StatusBadRequest)
		return
	}

	backend, ok := s.backends[req.Model]
	if !ok {
		http.Error(w, `{"error":"model not found"}`, http.StatusNotFound)
		return
	}

	resp, err := backend.Infer(r.Context(), inferenceRequestFromTTS(req))
	if err != nil {
		http.Error(w, `{"error":"inference failed"}`, http.StatusInternalServerError)
		return
	}

	format := req.ResponseFormat
	if format == "" {
		format = "wav"
	}

	switch format {
	case "wav":
		w.Header().Set("Content-Type", "audio/wav")
	case "mp3":
		w.Header().Set("Content-Type", "audio/mpeg")
	default:
		w.Header().Set("Content-Type", "audio/wav")
	}

	w.Write(resp.AudioData)
}
