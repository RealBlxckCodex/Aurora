package domain

type Voice struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Language  string `json:"language"`
	ModelID   string `json:"model_id"`
	Installed bool   `json:"installed"`
	Size      int64  `json:"size"`
	SHA256    string `json:"sha256,omitempty"`
	URL       string `json:"url,omitempty"`
}
