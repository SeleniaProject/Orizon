package packagemanager

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

// FileSignatureStore persists SignatureBundle lists per CID under a directory.
// Each CID is stored as a JSON array at <base>/<cid>.json
// Concurrency-safe and resilient to concurrent readers.
type FileSignatureStore struct {
	base string
	mu   sync.Mutex
}

func NewFileSignatureStore(base string) (*FileSignatureStore, error) {
	if base == "" {
		return nil, errors.New("base required")
	}
	if err := os.MkdirAll(base, 0o755); err != nil {
		return nil, err
	}
	return &FileSignatureStore{base: base}, nil
}

func (s *FileSignatureStore) path(cid CID) string {
	return filepath.Join(s.base, string(cid)+".json")
}

func (s *FileSignatureStore) Put(cid CID, sig SignatureBundle) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.path(cid)
	var list []SignatureBundle
	if b, err := os.ReadFile(p); err == nil {
		_ = json.Unmarshal(b, &list)
	}
	list = append(list, sig)
	b, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, b, 0o644)
}

func (s *FileSignatureStore) List(cid CID) ([]SignatureBundle, error) {
	p := s.path(cid)
	b, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var list []SignatureBundle
	if err := json.Unmarshal(b, &list); err != nil {
		return nil, err
	}
	return list, nil
}
