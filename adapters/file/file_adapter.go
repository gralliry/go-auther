// Package fileadapter provides a JSON file-backed adapter for policy persistence.
//
// Usage:
//
//	adapter := fileadapter.NewFileAdapter("policy.json")
//	a, _ := auther.NewAuthorizer(adapter)
package fileadapter

import (
	"encoding/json"
	"os"
	"sync"

	"auther"
)

// FileAdapter is a JSON file-backed adapter for Auther policy persistence.
// Writes are atomic via temp file + rename.
type FileAdapter struct {
	filePath string
	mu       sync.Mutex
}

// NewFileAdapter creates a new file adapter that persists to the given path.
func NewFileAdapter(filePath string) *FileAdapter {
	return &FileAdapter{filePath: filePath}
}

// Load reads the policy snapshot from the JSON file.
// Returns nil if the file does not exist.
func (fa *FileAdapter) Load() (*auther.PolicySnapshot, error) {
	fa.mu.Lock()
	defer fa.mu.Unlock()

	data, err := os.ReadFile(fa.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var snap auther.PolicySnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, err
	}
	return &snap, nil
}

// Save persists the policy snapshot to the JSON file.
// Uses atomic write: writes to temp file, then renames.
func (fa *FileAdapter) Save(snapshot *auther.PolicySnapshot) error {
	fa.mu.Lock()
	defer fa.mu.Unlock()

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}

	tmpPath := fa.filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmpPath, fa.filePath)
}
