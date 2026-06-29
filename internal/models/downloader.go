package models

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

type Downloader struct {
	client *http.Client
}

func NewDownloader() *Downloader {
	return &Downloader{
		client: &http.Client{},
	}
}

func (d *Downloader) Download(ctx context.Context, url, dest, expectedSHA256 string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	tmpPath := dest + ".tmp"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "Aurora/1.0")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("create tmp file: %w", err)
	}

	hash := sha256.New()
	writer := io.MultiWriter(f, hash)

	if _, err := io.Copy(writer, resp.Body); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write file: %w", err)
	}

	if err := f.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close file: %w", err)
	}

	if expectedSHA256 != "" {
		got := hex.EncodeToString(hash.Sum(nil))
		if got != expectedSHA256 {
			os.Remove(tmpPath)
			return fmt.Errorf("sha256 mismatch: expected %s, got %s", expectedSHA256, got)
		}
	}

	if err := os.Rename(tmpPath, dest); err != nil {
		return fmt.Errorf("rename: %w", err)
	}

	return nil
}
