package g2p

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

type Language string

const (
	LangDE Language = "de"
	LangEN Language = "en"
	LangFR Language = "fr"
	LangES Language = "es"
	LangIT Language = "it"
	LangPT Language = "pt"
	LangRU Language = "ru"
	LangJA Language = "ja"
	LangZH Language = "zh"
	LangKO Language = "ko"
)

type G2P struct {
	espeakVoice map[Language]string
}

func New() *G2P {
	return &G2P{
		espeakVoice: map[Language]string{
			LangDE: "de",
			LangEN: "en",
			LangFR: "fr",
			LangES: "es",
			LangIT: "it",
			LangPT: "pt",
			LangRU: "ru",
			LangJA: "ja",
			LangZH: "cmn",
			LangKO: "ko",
		},
	}
}

func (g *G2P) Convert(text string, lang Language) (string, error) {
	voice, ok := g.espeakVoice[lang]
	if !ok {
		voice = string(lang)
	}

	var out bytes.Buffer
	cmd := exec.Command("espeak-ng", "-q", "-v", voice, "--ipa", "-x", text)
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("espeak failed: %w", err)
	}

	phonemes := strings.TrimSpace(out.String())
	if phonemes == "" {
		return text, nil
	}

	return phonemes, nil
}

func (g *G2P) ConvertToIPA(text string, lang Language) (string, error) {
	voice, ok := g.espeakVoice[lang]
	if !ok {
		voice = string(lang)
	}

	var out bytes.Buffer
	cmd := exec.Command("espeak-ng", "-q", "-v", voice, "--ipa", "-x", text)
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("espeak failed: %w", err)
	}

	return strings.TrimSpace(out.String()), nil
}

func (g *G2P) PreprocessForTTS(text string, lang Language) string {
	phonemes, err := g.Convert(text, lang)
	if err != nil || phonemes == "" {
		return text
	}

	phonemes = strings.Join(strings.Fields(phonemes), " ")
	return phonemes
}

func DetectLanguage(text string) Language {
	scripts := map[Language]string{
		LangJA: "\\p{Hiragana}|\\p{Katakana}",
		LangZH: "\\p{Han}",
		LangKO: "\\p{Hangul}",
		LangRU: "\\p{Cyrillic}",
	}

	for lang, pattern := range scripts {
		if matchPattern(text, pattern) {
			return lang
		}
	}

	return LangEN
}

func matchPattern(text, pattern string) bool {
	cmd := exec.Command("grep", "-P", pattern)
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run() == nil
}
