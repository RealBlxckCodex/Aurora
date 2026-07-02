package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/RealBlxckCodex/Aurora/internal/inference"
)

type TTSRequest struct {
	Model          string  `json:"model"`
	Input          string  `json:"input"`
	Voice          string  `json:"voice"`
	ResponseFormat string  `json:"response_format"`
	Speed          float64 `json:"speed"`
	Stream         bool    `json:"stream"`
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

	format := req.ResponseFormat
	if format == "" {
		format = "wav"
	}

	if req.Stream {
		s.streamTTS(w, r, req, backend, format)
		return
	}

	resp, err := backend.Infer(r.Context(), inferenceRequestFromTTS(req))
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"inference failed: %s"}`, err.Error()), http.StatusInternalServerError)
		return
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

func (s *Server) streamTTS(w http.ResponseWriter, r *http.Request, req TTSRequest, backend inference.IInferenceBackend, format string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, `{"error":"streaming not supported"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	ctx := r.Context()

	if backend.SupportsStreaming() {
		ch, err := backend.Stream(ctx, inferenceRequestFromTTS(req))
		if err != nil {
			fmt.Fprintf(w, "event: error\ndata: {\"error\":\"%s\"}\n\n", err.Error())
			flusher.Flush()
			return
		}
		for chunk := range ch {
			select {
			case <-ctx.Done():
				return
			default:
			}
			encoded := base64.StdEncoding.EncodeToString(chunk.Data)
			jsonData, _ := json.Marshal(map[string]interface{}{
				"audio":     encoded,
				"format":    format,
				"timestamp": chunk.Timestamp.Seconds(),
				"final":     chunk.IsFinal,
			})
			fmt.Fprintf(w, "data: %s\n\n", jsonData)
			flusher.Flush()
			if chunk.IsFinal {
				return
			}
		}
		fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()
		return
	}

	resp, err := backend.Infer(ctx, inferenceRequestFromTTS(req))
	if err != nil {
		fmt.Fprintf(w, "event: error\ndata: {\"error\":\"%s\"}\n\n", err.Error())
		flusher.Flush()
		return
	}

	s.streamAudioChunks(w, flusher, resp.AudioData, format)
}

func (s *Server) streamAudioChunks(w http.ResponseWriter, flusher http.Flusher, audio []byte, format string) {
	chunkSize := 4096
	total := len(audio)
	sent := 0
	chunkIndex := 0
	totalChunks := (total + chunkSize - 1) / chunkSize

	for sent < total {
		end := sent + chunkSize
		if end > total {
			end = total
		}
		isFinal := end >= total

		encoded := base64.StdEncoding.EncodeToString(audio[sent:end])
		jsonData, _ := json.Marshal(map[string]interface{}{
			"audio":       encoded,
			"format":      format,
			"index":       chunkIndex,
			"total_chunks": totalChunks,
			"final":       isFinal,
		})
		fmt.Fprintf(w, "data: %s\n\n", jsonData)
		flusher.Flush()

		sent = end
		chunkIndex++
		time.Sleep(10 * time.Millisecond)
	}

	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()
}
