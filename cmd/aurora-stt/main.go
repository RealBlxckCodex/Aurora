package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/RealBlxckCodex/Aurora/internal/stt"
)

func main() {
	model := flag.String("model", "", "Path to STT model file")
	audio := flag.String("audio", "", "Input audio file path")
	output := flag.String("output", "", "Output transcription file (default: stdout)")
	language := flag.String("language", "", "Language code (e.g. de, en)")
	whisperBinary := flag.String("whisper-binary", "whisper.cpp", "Path to whisper.cpp binary")
	flag.Parse()

	if *model == "" || *audio == "" {
		fmt.Fprintf(os.Stderr, "Usage: aurora-stt --model <path> --audio <file> [--output out.txt] [--language de]\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	w := stt.NewWhisper(stt.WhisperConfig{
		BinaryPath: *whisperBinary,
		ModelPath:  *model,
		Language:   *language,
	})

	log.Printf("Transcribing with Whisper: model=%s audio=%s language=%s", *model, *audio, *language)
	transcription, err := w.Transcribe(ctx, *audio)
	if err != nil {
		log.Fatalf("STT failed: %v", err)
	}

	if *output != "" {
		if err := os.WriteFile(*output, []byte(transcription), 0644); err != nil {
			log.Fatalf("Failed to write output: %v", err)
		}
		log.Printf("Transcription written to %s", *output)
	} else {
		fmt.Print(transcription)
	}
}
