package cli

import (
	"fmt"

	"github.com/RealBlxckCodex/Aurora/internal/g2p"
	"github.com/spf13/cobra"
)

func NewG2PCmd() *cobra.Command {
	var (
		lang     string
		format string
	)

	cmd := &cobra.Command{
		Use:   "g2p <text>",
		Short: "Convert text to phonemes",
		Long: `Convert graphemes (text) to phonemes using espeak-ng.
Useful for TTS preprocessing and debugging phonemization.

Languages: de, en, fr, es, it, pt, ru, ja, zh, ko`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			text := args[0]
			langCode := g2p.Language(lang)

			g := g2p.New()

			switch format {
			case "ipa":
				result, err := g.ConvertToIPA(text, langCode)
				if err != nil {
					return fmt.Errorf("g2p failed: %w", err)
				}
				fmt.Println(result)

			case "phonemes":
				phonemes, err := g.Convert(text, langCode)
				if err != nil {
					return fmt.Errorf("g2p failed: %w", err)
				}
				fmt.Print("Input:  ")
				fmt.Println(text)
				fmt.Print("Phonemes: ")
				fmt.Println(phonemes)
				fmt.Print("Tokens: ")
				ids := tokenizePhonemes(phonemes)
				for i, id := range ids {
					if i > 0 {
						fmt.Print(" ")
					}
					fmt.Print(id)
				}
				fmt.Println()

			default:
				result, err := g.Convert(text, langCode)
				if err != nil {
					return fmt.Errorf("g2p failed: %w", err)
				}
				fmt.Println(result)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&lang, "lang", "l", "de", "Language code")
	cmd.Flags().StringVarP(&format, "format", "f", "phonemes", "Output format: phonemes, ipa")

	return cmd
}

// tokenizePhonemes re-exported for g2p CLI
func tokenizePhonemes(phonemes string) []int64 {
	return tokenizePhonemesInternal(phonemes)
}

func tokenizePhonemesInternal(phonemes string) []int64 {
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
