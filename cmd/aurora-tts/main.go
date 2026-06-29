package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/RealBlxckCodex/Aurora/internal/tts"
)

func main() {
	model := flag.String("model", "", "Path to TTS model file")
	text := flag.String("text", "", "Text to synthesize")
	output := flag.String("output", "output.wav", "Output audio file path")
	voice := flag.String("voice", "", "Voice identifier (e.g. de/female_1)")
	speed := flag.Float64("speed", 1.0, "Speed factor (0.5-2.0)")
	backend := flag.String("backend", "piper", "TTS backend: piper or kokoro")
	piperBinary := flag.String("piper-binary", "piper", "Path to piper binary")
	flag.Parse()

	if *model == "" || *text == "" {
		fmt.Fprintf(os.Stderr, "Usage: aurora-tts --model <path> --text <text> [--output out.wav] [--voice de/female_1] [--speed 1.0] [--backend piper|kokoro]\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	switch *backend {
	case "piper":
		p := tts.NewPiper(tts.PiperConfig{
			BinaryPath: *piperBinary,
			ModelPath:  *model,
			Voice:      *voice,
			Speed:      *speed,
		})

		log.Printf("Synthesizing with Piper: model=%s text=%q output=%s", *model, *text, *output)
		if err := p.Synthesize(ctx, *text, *output); err != nil {
			log.Fatalf("TTS failed: %v", err)
		}

	case "kokoro":
		k := tts.NewKokoro(tts.KokoroConfig{
			ModelPath: *model,
			VoicesDir: "",
			Voice:     *voice,
			Speed:     *speed,
		})

		if err := k.Initialize(); err != nil {
			log.Fatalf("Failed to initialize Kokoro: %v", err)
		}
		defer k.Close()

		log.Printf("Synthesizing with Kokoro: model=%s text=%q output=%s", *model, *text, *output)
		if err := k.Synthesize(ctx, *text, *output); err != nil {
			log.Fatalf("TTS failed: %v", err)
		}

	default:
		log.Fatalf("Unknown backend: %s", *backend)
	}

	log.Printf("Output written to %s", *output)
}
