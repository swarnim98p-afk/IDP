package projects

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

var ErrCursor = errors.New("invalid pagination cursor")

type MemoryStore struct {
	mu    sync.RWMutex
	items map[string]Project
	file  string
}

func NewMemoryStore() *MemoryStore { return &MemoryStore{items: make(map[string]Project)} }
func NewFileStore(path string) (*MemoryStore, error) {
	s := NewMemoryStore()
	s.file = path
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(data, &s.items); err != nil {
		return nil, fmt.Errorf("read project records: %w", err)
	}
	if s.items == nil {
		s.items = make(map[string]Project)
	}
	return s, nil
}
func (s *MemoryStore) Mode() string {
	if s.file != "" {
		return "file"
	}
	return "memory"
}
func (s *MemoryStore) Ready(context.Context) error { return nil }
func (s *MemoryStore) Put(ctx context.Context, p Project) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[p.ID]; ok {
		return ErrConflict
	}
	s.items[p.ID] = p
	if s.file != "" {
		if err := s.persist(); err != nil {
			delete(s.items, p.ID)
			return err
		}
	}
	return nil
}
func (s *MemoryStore) persist() error {
	data, err := json.MarshalIndent(s.items, "", "  ")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(s.file), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(s.file), "projects-*.tmp")
	if err != nil {
		return err
	}
	path := f.Name()
	defer os.Remove(path)
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(path, s.file)
}
func (s *MemoryStore) Get(ctx context.Context, id string) (Project, error) {
	if err := ctx.Err(); err != nil {
		return Project{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.items[id]
	if !ok {
		return Project{}, ErrNotFound
	}
	return p, nil
}
func (s *MemoryStore) List(ctx context.Context, limit int, cursor string) (Page, error) {
	if err := ctx.Err(); err != nil {
		return Page{}, err
	}
	lastID := ""
	if cursor != "" {
		decoded, err := base64.RawURLEncoding.DecodeString(cursor)
		if err != nil || len(decoded) == 0 {
			return Page{}, ErrCursor
		}
		lastID = string(decoded)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]Project, 0, len(s.items))
	for _, p := range s.items {
		items = append(items, p)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt == items[j].CreatedAt {
			return items[i].ID < items[j].ID
		}
		return items[i].CreatedAt > items[j].CreatedAt
	})
	start := 0
	if lastID != "" {
		found := false
		for i, p := range items {
			if p.ID == lastID {
				start = i + 1
				found = true
				break
			}
		}
		if !found {
			return Page{}, ErrCursor
		}
	}
	end := start + limit
	if end > len(items) {
		end = len(items)
	}
	page := Page{Projects: append([]Project{}, items[start:end]...)}
	if end < len(items) {
		page.NextCursor = base64.RawURLEncoding.EncodeToString([]byte(items[end-1].ID))
	}
	return page, nil
}
