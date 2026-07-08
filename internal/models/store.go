package models

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/RealBlxckCodex/Aurora/pkg/domain"
)

type Store struct {
	modelsDir string
	mu        sync.RWMutex
	cache     map[string]*domain.Model
}

func NewStore(modelsDir string) *Store {
	return &Store{
		modelsDir: modelsDir,
		cache:     make(map[string]*domain.Model),
	}
}

func (s *Store) dbPath() string {
	return filepath.Join(s.modelsDir, "aurora.db.json")
}

func (s *Store) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.dbPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read db: %w", err)
	}

	var models []*domain.Model
	if err := json.Unmarshal(data, &models); err != nil {
		return fmt.Errorf("parse db: %w", err)
	}

	for _, m := range models {
		s.cache[m.ID] = m
	}
	return nil
}

func (s *Store) List() []*domain.Model {
	s.mu.RLock()
	defer s.mu.RUnlock()

	models := make([]*domain.Model, 0, len(s.cache))
	for _, m := range s.cache {
		models = append(models, m)
	}
	return models
}

func (s *Store) Get(id string) *domain.Model {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cache[id]
}

func (s *Store) Add(model *domain.Model) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache[model.ID] = model
	return s.save()
}

func (s *Store) Remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.cache, id)
	return s.save()
}

func (s *Store) save() error {
	models := make([]*domain.Model, 0, len(s.cache))
	for _, m := range s.cache {
		models = append(models, m)
	}

	data, err := json.MarshalIndent(models, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(s.modelsDir, 0755); err != nil {
		return fmt.Errorf("create models dir: %w", err)
	}

	return os.WriteFile(s.dbPath(), data, 0644)
}

func (s *Store) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.save()
}

func (s *Store) ModelPath(model *domain.Model) string {
	ext := string(model.Format)
	return filepath.Join(s.modelsDir, model.ID, "model."+ext)
}
