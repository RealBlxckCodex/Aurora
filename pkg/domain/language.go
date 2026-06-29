package domain

type LanguagePack struct {
	Code       string   `json:"code"`
	Name       string   `json:"name"`
	NativeName string   `json:"native_name"`
	TTSModels  []string `json:"tts_models"`
	STTModels  []string `json:"stt_models"`
	Voices     []Voice  `json:"voices"`
	Size       int64    `json:"size"`
}
