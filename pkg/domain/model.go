package domain

type ModelType string

const (
	ModelTypeTTS ModelType = "tts"
	ModelTypeSTT ModelType = "stt"
)

type ModelFormat string

const (
	ModelFormatGGUF ModelFormat = "gguf"
	ModelFormatONNX ModelFormat = "onnx"
	ModelFormatBin  ModelFormat = "bin"
)

type Model struct {
	ID           string      `json:"id"`
	Name         string      `json:"name"`
	Type         ModelType   `json:"type"`
	Format       ModelFormat `json:"format"`
	Version      string      `json:"version"`
	Size         int64       `json:"size"`
	Quantization string      `json:"quantization,omitempty"`
	SHA256       string      `json:"sha256"`
	URL          string      `json:"url"`
	Installed    bool        `json:"installed"`
	Backend      string      `json:"backend"`
	Languages    []string    `json:"languages"`
}
