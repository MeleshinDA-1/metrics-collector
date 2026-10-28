package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

type FileObserver struct {
	mu   sync.Mutex
	file *os.File
}

func NewFileObserver(path string) (*FileObserver, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil, fmt.Errorf("open audit file: %w", err)
	}
	return &FileObserver{file: file}, nil
}

func (o *FileObserver) Notify(_ context.Context, event Event) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if err := json.NewEncoder(o.file).Encode(event); err != nil {
		return fmt.Errorf("append audit event: %w", err)
	}
	return nil
}

func (o *FileObserver) Close() error {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.file.Close()
}
