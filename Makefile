BINARY_DIR := bin
AURORA_BINARY := $(BINARY_DIR)/aurora
TTS_BINARY := $(BINARY_DIR)/aurora-tts
STT_BINARY := $(BINARY_DIR)/aurora-stt
GO := go
LDFLAGS := -ldflags="-X main.Version=$(shell git describe --tags --always --dirty 2>/dev/null || echo '1.0.0')"

.PHONY: all build clean test lint run-tts run-stt

all: build

build: $(AURORA_BINARY) $(TTS_BINARY) $(STT_BINARY)

$(AURORA_BINARY): cmd/aurora/main.go internal/cli/*.go internal/api/*.go internal/inference/*.go internal/hardware/*.go internal/models/*.go internal/config/*.go pkg/domain/*.go
	mkdir -p $(BINARY_DIR)
	$(GO) build $(LDFLAGS) -o $@ ./cmd/aurora

$(TTS_BINARY): cmd/aurora-tts/main.go internal/tts/*.go
	mkdir -p $(BINARY_DIR)
	$(GO) build $(LDFLAGS) -o $@ ./cmd/aurora-tts

$(STT_BINARY): cmd/aurora-stt/main.go internal/stt/*.go
	mkdir -p $(BINARY_DIR)
	$(GO) build $(LDFLAGS) -o $@ ./cmd/aurora-stt

clean:
	rm -rf $(BINARY_DIR)

test:
	$(GO) test ./...

lint:
	$(GO) vet ./...
	@which golint 2>/dev/null && golint ./... || true

run-tts:
	$(TTS_BINARY) --model models/piper/de_DE-thorsten-medium.onnx --text "Hallo Welt" --output out.wav

run-stt:
	$(STT_BINARY) --model models/whisper/ggml-turbo.bin --audio in.wav --output out.txt
